# Templates

Sink configuration values like `partitionKey`, `payload`, `body`, `url`, `topic`,
header values, etc. are evaluated as Go [`text/template`](https://pkg.go.dev/text/template)
strings. A value with no `{{ ... }}` action is a plain string and is passed
through verbatim; anything containing a valid action is rendered for every
change-event before the sink is invoked.

This document covers:

- [Where templates run](#where-templates-run)
- [Render data context](#render-data-context)
- [Static vs. dynamic strings](#static-vs-dynamic-strings)
- [Template syntax](#template-syntax)
- [Function reference](#function-reference)
- [Recipes](#recipes)

## Where templates run

Every string field in the sink config that documents "text/template" support is
parsed at config-load time and rendered on every WAL event. This includes:

- `kafka.connection`, `kafka.topic`, `kafka.partitionKey`, `kafka.payload`
- `http.url`, `http.method`, `http.body`, each value in `http.headers`

If a string contains no `{{` token (or contains `{{` but fails to parse as a
template), it is treated as a static string and rendered as-is on every event.
Otherwise it is parsed once and re-rendered per event with the row data.

## Render data context

Each render receives a `*sinks.Context` value as the dot (`.`):

| Field        | Type             | When populated                                  |
| ------------ | ---------------- | ----------------------------------------------- |
| `.Schema`    | `string`         | always                                          |
| `.Table`     | `string`         | always                                          |
| `.Operation` | `string`         | one of `insert`, `update`, `delete`, `truncate` |
| `.Prev`      | `map[string]any` | `update`, `delete`                              |
| `.Next`      | `map[string]any` | `insert`, `update`                              |

Column values live under `.Prev` / `.Next`, keyed by column name:

```yaml
partitionKey: "{{ .Next.id }}"
payload: |
  {{ .Next | json false }}
```

When the column name is not a valid Go identifier (contains spaces, hyphens, or
starts with a digit), use `index`:

```yaml
partitionKey: '{{ index .Next "user-id" }}'
```

`truncate` events carry empty `.Prev` and `.Next`. Guard with `if`/`with` if a
template needs them:

```yaml
payload: |
  {{ with .Next }}{{ .id }}{{ else }}truncated{{ end }}
```

A column the source could not deliver is *unavailable*. On a table whose replica
identity is not `FULL`, that covers an unchanged TOAST value on `update`, and
every non-key column of the old row on `delete` or on an `update` that changes a
key column. Such a column renders as an empty string, compares equal to `""`,
and counts as empty for `empty`, `coalesce` and `required`; `unavailable` is the
only way to tell it apart from a column that really is `''`:

```yaml
payload: |
  {{ if unavailable .Next.blob }}unchanged{{ else }}{{ .Next.blob }}{{ end }}
```

## Static vs. dynamic strings

The parser classifies each string at load time:

- **static** — no `{{` token, or `{{` present but unparseable. Rendered to the
  literal source. Errors in malformed templates are _not_ surfaced; the value
  is kept as-is.
- **dynamic** — contains a valid `{{ ... }}` action. Re-rendered per event
  against the data context. Render errors abort sink delivery for that event.

This means a typo like `{{ .Next.id` (missing `}}`) is silently treated as a
literal string. Validate dynamic strings by running the config and watching for
either the literal output or a render error in the logs.

## Template syntax

Quick reference for the `text/template` constructs used most often. See the
[stdlib docs](https://pkg.go.dev/text/template) for the full grammar.

### Field access and pipelines

```
{{ .Next.id }}                      access a field
{{ .Next.id | printf "%05d" }}      pipe value through a function
{{ printf "%s/%s" .Schema .Table }} call a function with multiple args
```

### Variables

```
{{ $id := .Next.id }}{{ $id }}-{{ $id }}
```

### Conditionals

```
{{ if eq .Operation "delete" }}DEL{{ else }}UP{{ end }}
{{ if .Next.email }}{{ .Next.email }}{{ else }}<no-email>{{ end }}
```

### Iteration

```
{{ range $k, $v := .Next }}{{ $k }}={{ $v }} {{ end }}
{{ range .Next.tags }}- {{ . }}
{{ end }}
```

### Whitespace trim

`{{- ...` and `... -}}` consume preceding/trailing whitespace including
newlines. Useful inside `range`/`if` blocks to keep multi-line payloads tidy:

```
{
{{- range $k, $v := .Next }}
  "{{ $k }}": {{ $v | json false }},
{{- end }}
}
```

### Comments

```
{{/* this is a template comment, not emitted */}}
```

## Function reference

All function names are lowercase. Where a helper takes a "string" argument, a
nil/missing field is coerced to `""` (no error).

### Standard `text/template` builtins

| Name                | Purpose                                                         |
| ------------------- | --------------------------------------------------------------- |
| `and`               | Returns the first false argument, or the last if all are truthy |
| `or`                | Returns the first truthy argument, or the last if all false     |
| `not`               | Boolean negation                                                |
| `eq`                | `eq a b [c …]` — true if `a == b` (or any of the rest)          |
| `ne`                | Not equal                                                       |
| `lt`/`le`/`gt`/`ge` | Numeric / string comparisons                                    |
| `len`               | Length of slice, map, string, channel                           |
| `index`             | `index x i [j …]` — `x[i][j]…`                                  |
| `slice`             | `slice x i j [k]` — `x[i:j]` or `x[i:j:k]`                      |
| `call`              | Invoke a function value passed via the data context             |
| `print`             | `fmt.Sprint`                                                    |
| `printf`            | `fmt.Sprintf`                                                   |
| `println`           | `fmt.Sprintln`                                                  |
| `html`              | HTML-escape                                                     |
| `js`                | JavaScript-string-escape                                        |
| `urlquery`          | Percent-encode for URL query                                    |

### Flow control

| Name       | Signature                       | Notes                                                                     |
| ---------- | ------------------------------- | ------------------------------------------------------------------------- |
| `empty`    | `empty v` → bool                | true for nil, "", 0, false, empty container                               |
| `coalesce` | `coalesce v1 v2 …`              | First non-empty (or nil if all empty)                                     |
| `default`  | `default fallback v`            | Returns `v` unless it is _typed nil_ — empty strings and `0` pass through |
| `ternary`  | `ternary trueVal falseVal cond` | `cond ? trueVal : falseVal`                                               |
| `required` | `required msg v`                | Returns `v`, or aborts the render with `msg` if empty                     |
| `fail`     | `fail msg`                      | Always aborts the render with `msg`                                       |
| `unavailable` | `unavailable v` → bool       | true for a column the source could not deliver                            |

### Case conversion

All tokenize the input on word boundaries (case changes, digits, separators).

| Name              | Example input | Example output                             |
| ----------------- | ------------- | ------------------------------------------ |
| `camelcase`       | `foo bar`     | `fooBar`                                   |
| `pascalcase`      | `foo bar`     | `FooBar`                                   |
| `snakecase`       | `fooBar`      | `foo_bar`                                  |
| `constantcase`    | `fooBar`      | `FOO_BAR`                                  |
| `kebabcase`       | `fooBar`      | `foo-bar`                                  |
| `headercase`      | `fooBar`      | `Foo-Bar`                                  |
| `dotcase`         | `fooBar`      | `foo.bar`                                  |
| `pathcase`        | `fooBar`      | `foo/bar`                                  |
| `pascalsnakecase` | `fooBar`      | `Foo_Bar`                                  |
| `capitalcase`     | `foo bar`     | `Foo Bar`                                  |
| `upper`           | `fooBar`      | `FOO BAR`                                  |
| `lower`           | `FooBar`      | `foo bar`                                  |
| `upperfirst`      | `hello`       | `Hello` (only first rune; no tokenization) |
| `lowerfirst`      | `Hello`       | `hello` (only first rune; no tokenization) |

### Strings

| Name         | Signature                                                                 |
| ------------ | ------------------------------------------------------------------------- |
| `trim`       | `trim s`                                                                  |
| `trimall`    | `trimall cutset s` — trim any char in `cutset` from both ends             |
| `trimprefix` | `trimprefix prefix s`                                                     |
| `trimsuffix` | `trimsuffix suffix s`                                                     |
| `nospace`    | Strip all Unicode whitespace                                              |
| `swapcase`   | Invert case of every cased rune                                           |
| `substr`     | `substr start end s` — rune indices, negative `end` counts from the right |
| `trunc`      | `trunc n s` — first/last `n` runes (negative counts from end)             |
| `abbrev`     | `abbrev width s` — truncate with `…`-style suffix                         |
| `abbrevboth` | `abbrevboth offset maxWidth s`                                            |
| `initials`   | First rune of each whitespace-separated word                              |
| `wrap`       | `wrap width s` — word-wrap with `\n`                                      |
| `wrapwith`   | `wrapwith width sep s`                                                    |
| `contains`   | `contains sub s`                                                          |
| `hasprefix`  | `hasprefix prefix s`                                                      |
| `hassuffix`  | `hassuffix suffix s`                                                      |
| `replace`    | `replace old new s`                                                       |
| `repeat`     | `repeat count s`                                                          |
| `cat`        | Space-join args after stringifying                                        |
| `quote`      | Wrap each arg in double quotes, space-joined                              |
| `squote`     | Same, with single quotes                                                  |
| `indent`     | `indent n s` — prepend `n` spaces to every line                           |
| `nindent`    | `nindent n s` — leading newline + `indent`                                |
| `plural`     | `plural one many count`                                                   |

### Random strings

| Name           | Signature        | Output                                         |
| -------------- | ---------------- | ---------------------------------------------- |
| `randalpha`    | `randalpha n`    | `n` random ASCII letters                       |
| `randalphanum` | `randalphanum n` | Letters + digits                               |
| `randnumeric`  | `randnumeric n`  | Digits                                         |
| `randascii`    | `randascii n`    | Printable ASCII (32–126)                       |
| `randbytes`    | `randbytes n`    | `n` cryptographic random bytes, base64-encoded |
| `shuffle`      | `shuffle s`      | `s` with runes randomly reordered              |

`randalpha`/`randalphanum`/`randnumeric`/`randascii` use `math/rand` and are
**not** cryptographically secure. Use `randbytes` when you need that property.

### Type conversion

| Name            | Signature         | Behavior                                                              |
| --------------- | ----------------- | --------------------------------------------------------------------- |
| `atoi`          | `atoi s` → int    | Bad input → `0`                                                       |
| `int`/`int64`   | `int v`           | Coerces numbers, bools, and parseable strings; otherwise `0`          |
| `float64`       | `float64 v`       | Same coercion rules                                                   |
| `todecimal`     | `todecimal v`     | Parses `v` as **octal** and returns the decimal value (matches sprig) |
| `tostring`      | `tostring v`      | `fmt.Sprint(v)`                                                       |
| `tostrings`     | `tostrings list`  | Stringify every element of a `[]string`/`[]any`                       |
| `json`          | `json prettify v` | Marshal to JSON; `prettify=true` uses 2-space indent                  |
| `fromjson`      | `fromjson s`      | Parse JSON object → `map[string]any`                                  |
| `fromjsonarray` | `fromjsonarray s` | Parse JSON array → `[]any`                                            |

### YAML

| Name            | Signature         | Behavior                                     |
| --------------- | ----------------- | -------------------------------------------- |
| `yaml`          | `yaml prettify v` | Marshal; `prettify=true` uses 4-space indent |
| `fromyaml`      | `fromyaml s`      | Parse YAML scalar/document                   |
| `fromyamlarray` | `fromyamlarray s` | Parse YAML array → `[]any`                   |

### Regex (`regexp` package syntax)

| Name                     | Signature                                         |
| ------------------------ | ------------------------------------------------- |
| `regexmatch`             | `regexmatch pattern s` → bool                     |
| `regexfind`              | `regexfind pattern s` → first match or `""`       |
| `regexfindall`           | `regexfindall pattern n s` → slice (`n=-1` = all) |
| `regexreplaceall`        | `regexreplaceall pattern repl s` (`$1` expansion) |
| `regexreplaceallliteral` | Same, no `$` expansion                            |
| `regexsplit`             | `regexsplit pattern n s`                          |

Compile errors abort the render.

### Encoding

| Name     | Behavior                     |
| -------- | ---------------------------- |
| `b64enc` | Base64 (std) encode          |
| `b64dec` | Base64 decode; errors → `""` |
| `b32enc` | Base32 (std) encode          |
| `b32dec` | Base32 decode; errors → `""` |

### Hashes & crypto

| Name         | Signature                       | Notes                                                              |
| ------------ | ------------------------------- | ------------------------------------------------------------------ |
| `sha1`        | `sha1 s` → hex digest             |                                                                    |
| `sha256`      | `sha256 s` → hex digest           |                                                                    |
| `adler32`     | `adler32 s` → decimal string      |                                                                    |
| `aes_encrypt` | `aes_encrypt key plaintext`       | AES-256-CBC; `key` is SHA-256-derived. IV-prefixed, base64-encoded |
| `aes_decrypt` | `aes_decrypt key b64`             | Reverses `aes_encrypt`                                             |

`sha1` is for non-security uses (cache keys, dedupe). Don't use it for
authentication.

### Date & time

`asTime`-compatible inputs: `time.Time`, `*time.Time`, int/int64 (Unix
seconds), or RFC3339 strings.

| Name             | Signature                                              |
| ---------------- | ------------------------------------------------------ |
| `now`            | `now` → current `time.Time`                            |
| `date`           | `date layout v`                                        |
| `dateinzone`     | `dateinzone layout v zone`                             |
| `htmldate`       | `htmldate v` → `YYYY-MM-DD`                            |
| `htmldateinzone` | `htmldateinzone v zone`                                |
| `unixepoch`      | `unixepoch v` → string of Unix seconds                 |
| `ago`            | `ago v` → Go duration string                           |
| `duration`       | `duration secs` (int or string)                        |
| `durationround`  | Round to most significant unit (e.g. `2h`, `3d`, `1y`) |
| `todate`         | `todate layout s` → `time.Time`                        |
| `datemodify`     | `datemodify spec v` → `time.Time` (e.g. `+1.5h`)       |

`layout` uses Go reference layout: `2006-01-02 15:04:05`.

Bad inputs to `date`, `unixepoch`, `ago`, `durationround`, `dateinzone`,
`htmldate`, `htmldateinzone`, `duration` return `""` instead of erroring.
`todate` and `datemodify` propagate parse errors.

### Maps / dicts

| Name             | Signature                                                  |
| ---------------- | ---------------------------------------------------------- |
| `dict`           | `dict k1 v1 [k2 v2 …]` → `map[string]any` (odd args error) |
| `get`            | `get d key`                                                |
| `set`            | `set d key value` → returns `d`                            |
| `unset`          | `unset d key` → returns `d`                                |
| `haskey`         | `haskey d key`                                             |
| `keys`           | `keys d` → sorted `[]string`                               |
| `values`         | `values d` → values in key-sorted order                    |
| `pick`           | `pick d k1 k2 …` → subset map                              |
| `pluck`          | `pluck key d1 d2 …` → present values across dicts          |
| `dig`            | `dig k1 k2 … fallback d` — nested lookup with fallback     |
| `merge`          | Shallow merge; first map wins                              |
| `mergeoverwrite` | Shallow merge; later maps win                              |
| `omit`           | `omit d k1 k2 …` — case-insensitive key removal            |
| `deepcopy`       | Recursively copy nested `map[string]any`/`[]any`           |

### Lists

| Name             | Signature                                  |
| ---------------- | ------------------------------------------ |
| `list`           | `list v1 v2 …` → `[]any`                   |
| `first`/`last`   | First / last element (nil if empty)        |
| `rest`/`initial` | All but first / all but last               |
| `append`         | `append xs v…`                             |
| `prepend`        | `prepend xs v…`                            |
| `concat`         | `concat xs ys …`                           |
| `reverse`        | Reverse a list                             |
| `uniq`           | Remove duplicates (first wins)             |
| `compact`        | Remove empty/zero values (per `empty`)     |
| `without`        | `without xs v…` — drop matching elements   |
| `has`            | `has v xs` → bool                          |
| `chunk`          | `chunk size xs` → `[][]any`                |
| `until`          | `until n` → `[0..n-1]`                     |
| `untilstep`      | `untilstep start end step`                 |
| `seq`            | `seq n` / `seq a b` / `seq start step end` |

### Math (integer)

`add`, `add1`, `sub`, `mul`, `div`, `mod`, `max`, `min`. Variadic where
appropriate; division/modulo by zero return `0`.

### Math (float)

`addf`, `add1f`, `subf`, `mulf`, `divf`, `maxf`, `minf`, `floor`, `ceil`,
`round`. `round v precision` rounds to `precision` decimals.

### File path

| Name    | Behavior                             |
| ------- | ------------------------------------ |
| `base`  | `path.Base`                          |
| `dir`   | `path.Dir`                           |
| `clean` | `path.Clean`                         |
| `ext`   | File extension including the dot     |
| `isabs` | Reports whether the path is absolute |

### URL

| Name       | Signature                                                                                                 |
| ---------- | --------------------------------------------------------------------------------------------------------- |
| `urlparse` | `urlparse raw` → map with `scheme`, `host`, `hostname`, `path`, `query`, `opaque`, `fragment`, `userinfo` |
| `urljoin`  | `urljoin map` — inverse of `urlparse`                                                                     |

### Identifiers

| Name     | Behavior                |
| -------- | ----------------------- |
| `uuidv4` | Random RFC 4122 v4 UUID |

### Process environment

| Name  | Behavior                                                                                |
| ----- | --------------------------------------------------------------------------------------- |
| `env` | `env "VAR"` — `os.Getenv`. Read at render time, so updates are picked up between events |

## Recipes

### JSON payload of the new row

```yaml
payload: |
  {{ .Next | json true }}
```

### Routing key based on the operation

```yaml
partitionKey: '{{ if eq .Operation "delete" }}{{ .Prev.id }}{{ else }}{{ .Next.id }}{{ end }}'
```

### CDC envelope with metadata

```yaml
payload: |
  {
    "schema": "{{ .Schema }}",
    "table": "{{ .Table }}",
    "op": "{{ .Operation }}",
    "ts": "{{ now | htmldate }}",
    "before": {{ .Prev | json false }},
    "after":  {{ .Next | json false }}
  }
```

### Stable hash for sharding

```yaml
partitionKey: '{{ printf "%s:%v" .Table .Next.id | sha256 }}'
```

### Conditional URL by table

```yaml
url: "https://api.example.com/{{ .Table | snakecase }}/{{ .Next.id }}"
```

### Per-event auth header from env

```yaml
headers:
  Authorization: 'Bearer {{ env "API_TOKEN" }}'
```

### Required field guard

```yaml
partitionKey: '{{ required "id is required" .Next.id }}'
```

### Idempotency key

```yaml
headers:
  Idempotency-Key: '{{ printf "%s/%s/%v/%s" .Schema .Table .Next.id .Operation | sha1 }}'
```

## Reference

Per-function reference with a minimal example (input data → template → output)
for every helper registered in the `Funcs` map. Output is shown after `→`.
"Input" is included only when the function reads from the data context.

### Standard `text/template` builtins

#### `and`

Returns the first falsy argument, or the last argument when all are truthy.

```
{{ and "x" "y" "z" }} → z
{{ and "x" "" "z" }} →
```

#### `or`

Returns the first truthy argument, or the last when all are falsy.

```
{{ or "" 0 "first" }} → first
{{ or "" 0 false }} → false
```

#### `not`

Boolean negation of a value's truthiness.

```
{{ not false }} → true
{{ not "x" }} → false
```

#### `eq`

`eq a b [c …]` — true if `a` equals any of the rest.

```
{{ eq 1 1 }} → true
{{ eq "x" "y" "x" }} → true
```

#### `ne`

Negation of `eq` (two-arg).

```
{{ ne 1 2 }} → true
```

#### `lt` / `le` / `gt` / `ge`

Two-arg numeric/string comparisons.

```
{{ lt 1 2 }} → true
{{ le 2 2 }} → true
{{ gt 3 2 }} → true
{{ ge 2 2 }} → true
```

#### `len`

Length of slice, map, string, channel, or array.

```
{{ len "hello" }} → 5
{{ len (list 1 2 3) }} → 3
```

#### `index`

`index x i [j …]` — equivalent to `x[i][j]…`. Works on slices, arrays, strings,
and maps.

```
{{ index (list "a" "b" "c") 1 }} → b
{{ index (dict "a" (dict "b" "deep")) "a" "b" }} → deep
```

#### `slice`

`slice x i j [k]` — `x[i:j]` or three-arg `x[i:j:k]`.

```
{{ slice "hello" 1 4 }} → ell
{{ slice (list 1 2 3 4) 1 3 }} → [2 3]
```

#### `call`

Invoke a function value passed via the data context.

Input: `.Add = func(a, b int) int { return a + b }`

```
{{ call .Add 2 3 }} → 5
```

#### `print` / `printf` / `println`

Direct passthroughs of `fmt.Sprint`, `fmt.Sprintf`, `fmt.Sprintln`.

```
{{ print "a" "b" }} → ab
{{ printf "%05d" 42 }} → 00042
{{ println "x" }} → x\n
```

#### `html`

HTML-escape special characters.

```
{{ html "<a>&amp;</a>" }} → &lt;a&gt;&amp;amp;&lt;/a&gt;
```

#### `js`

JavaScript-string escape.

```
{{ js "a'b" }} → a\'b
```

#### `urlquery`

Percent-encode a value for URL query strings.

```
{{ urlquery "a b" }} → a+b
```

### Flow control

#### `empty`

Reports whether a value is the zero value for its type (nil, "", `0`, `false`,
empty container, nil pointer).

```
{{ empty "" }} → true
{{ empty "x" }} → false
{{ empty 0 }} → true
```

#### `coalesce`

Returns the first non-empty argument (per `empty`), or nothing if all are empty.

```
{{ coalesce "" 0 nil "first" "second" }} → first
```

#### `default`

`default fallback v` — returns `v` unless it is _typed nil_. Empty strings and
zero numbers pass through unchanged.

```
{{ default "fb" .Missing }} → fb
{{ default "fb" "" }} →
{{ default "fb" "x" }} → x
```

#### `ternary`

`ternary trueVal falseVal cond` — pipeline-friendly conditional.

```
{{ ternary "y" "n" true }} → y
{{ ternary "y" "n" false }} → n
```

#### `required`

`required msg v` — returns `v`, or aborts rendering with `msg` if `v` is empty.

```
{{ required "id is required" "ok" }} → ok
{{ required "id is required" "" }} → render error: id is required
```

#### `unavailable`

`unavailable v` — reports whether `v` is a column the source could not deliver.
Such a column is indistinguishable from `""` everywhere else: it renders empty,
compares equal to `""`, and counts as empty for `empty`, `coalesce` and
`required`.

```
{{ unavailable .Next.blob }} → true
{{ unavailable .Next.id }} → false
{{ unavailable "" }} → false
```

#### `fail`

Aborts rendering with the given message. Useful in branches that should be
unreachable.

```
{{ if .Bad }}{{ fail "bad input" }}{{ end }} → render error: bad input
```

### Case conversion

All case helpers tokenize the input on word boundaries (case changes, digit
boundaries, non-alphanumerics) before re-joining.

#### `camelcase`

```
{{ "get URL path" | camelcase }} → getUrlPath
```

#### `pascalcase`

```
{{ "get url path" | pascalcase }} → GetUrlPath
```

#### `snakecase`

```
{{ "GetURLPath" | snakecase }} → get_url_path
```

#### `constantcase`

```
{{ "fooBar" | constantcase }} → FOO_BAR
```

#### `kebabcase`

```
{{ "fooBar" | kebabcase }} → foo-bar
```

#### `headercase`

Title-cased words joined by `-` (a.k.a. Train-Case).

```
{{ "fooBar" | headercase }} → Foo-Bar
```

#### `dotcase`

```
{{ "fooBar" | dotcase }} → foo.bar
```

#### `pathcase`

```
{{ "fooBar" | pathcase }} → foo/bar
```

#### `pascalsnakecase`

Title-cased words joined by `_`.

```
{{ "fooBar" | pascalsnakecase }} → Foo_Bar
```

#### `capitalcase`

Title-cased words joined by spaces.

```
{{ "foo bar baz" | capitalcase }} → Foo Bar Baz
```

#### `upper`

Tokenize, uppercase each word, join with spaces.

```
{{ "fooBar" | upper }} → FOO BAR
```

#### `lower`

Tokenize, lowercase each word, join with spaces.

```
{{ "FooBar" | lower }} → foo bar
```

#### `upperfirst`

Uppercases only the first rune; the rest is unchanged. No tokenization.

```
{{ "hello" | upperfirst }} → Hello
```

#### `lowerfirst`

Lowercases only the first rune.

```
{{ "Hello" | lowerfirst }} → hello
```

### Strings

#### `trim`

Strip leading and trailing whitespace.

```
{{ trim "  hi  " }} → hi
```

#### `trimall`

`trimall cutset s` — trim any rune in `cutset` from both ends.

```
{{ trimall "xy" "xyhelloyx" }} → hello
```

#### `trimprefix`

```
{{ trimprefix "foo" "foobar" }} → bar
```

#### `trimsuffix`

```
{{ trimsuffix "bar" "foobar" }} → foo
```

#### `nospace`

Remove every Unicode whitespace rune.

```
{{ nospace "a b\tc\nd" }} → abcd
```

#### `swapcase`

Invert case of every cased rune.

```
{{ swapcase "HelloWorld" }} → hELLOwORLD
```

#### `substr`

`substr start end s` — rune-indexed substring. Negative `end` counts from the
right; bounds are clamped.

```
{{ substr 1 4 "hello" }} → ell
{{ substr 1 -1 "hello" }} → ell
```

#### `trunc`

`trunc n s` — keep the first `n` runes, or the last `-n` if `n` is negative.

```
{{ trunc 3 "abcdef" }} → abc
{{ trunc -3 "abcdef" }} → def
```

#### `abbrev`

`abbrev width s` — truncate to `width` runes ending with `...` if shortened.
`width` must be ≥4.

```
{{ abbrev 8 "abcdefghij" }} → abcde...
```

#### `abbrevboth`

`abbrevboth offset maxWidth s` — abbreviate from both sides.

```
{{ abbrevboth 5 10 "abcdefghijklmnop" }} → ...fghi...
```

#### `initials`

First rune of each whitespace-separated word.

```
{{ initials "John Ronald Reuel" }} → JRR
```

#### `wrap`

`wrap width s` — word-wrap with `\n`.

```
{{ wrap 5 "hello world foo" }} → hello\nworld\nfoo
```

#### `wrapwith`

`wrapwith width sep s` — word-wrap with a custom separator.

```
{{ wrapwith 5 " | " "hello world" }} → hello | world
```

#### `contains`

```
{{ contains "ell" "hello" }} → true
```

#### `hasprefix`

```
{{ hasprefix "foo" "foobar" }} → true
```

#### `hassuffix`

```
{{ hassuffix "bar" "foobar" }} → true
```

#### `replace`

`replace old new s` — replace every occurrence.

```
{{ replace "a" "X" "banana" }} → bXnXnX
```

#### `repeat`

`repeat count s` — concat `s` `count` times. Non-positive `count` returns "".

```
{{ repeat 3 "ab" }} → ababab
```

#### `cat`

Stringify each arg and join with single spaces.

```
{{ cat "a" 1 true }} → a 1 true
```

#### `quote`

Wrap each arg in double quotes, space-joined.

```
{{ quote "a" "b" }} → "a" "b"
```

#### `squote`

Wrap each arg in single quotes, space-joined.

```
{{ squote "a" "b" }} → 'a' 'b'
```

#### `indent`

`indent n s` — prepend `n` spaces to every line, including the first.

```
{{ indent 2 "a\nb" }} → "  a\n  b"
```

#### `nindent`

Like `indent` but with a leading newline.

```
{{ nindent 2 "a\nb" }} → "\n  a\n  b"
```

#### `plural`

`plural one many count` — pick `one` if `count == 1`, else `many`.

```
{{ plural "apple" "apples" 1 }} → apple
{{ plural "apple" "apples" 2 }} → apples
```

### Random strings

#### `randalpha`

`randalpha n` — `n` random ASCII letters (math/rand).

```
{{ randalpha 8 }} → e.g. KdwQpAzM
```

#### `randalphanum`

`n` random ASCII letters + digits.

```
{{ randalphanum 8 }} → e.g. K4w2pAzM
```

#### `randnumeric`

`n` random ASCII digits.

```
{{ randnumeric 6 }} → e.g. 482910
```

#### `randascii`

`n` random printable ASCII chars (32–126).

```
{{ randascii 6 }} → e.g. )^k!2#
```

#### `shuffle`

Reorder runes of a string randomly.

```
{{ shuffle "abcdef" }} → e.g. cabfed
```

### Type conversion

#### `atoi`

Parse base-10 int; bad input returns `0`.

```
{{ atoi "42" }} → 42
{{ atoi "abc" }} → 0
```

#### `int` / `int64`

Coerce numbers, bools, and parseable strings; otherwise `0`.

```
{{ int "12" }} → 12
{{ int 3.7 }} → 3
{{ int true }} → 1
```

#### `float64`

Coerce to float64; unparseable strings yield `0`.

```
{{ float64 "2.5" }} → 2.5
```

#### `todecimal`

Parse the input as **octal** (Go syntax) and return its decimal value.

```
{{ todecimal "777" }} → 511
```

#### `tostring`

`fmt.Sprint(v)`.

```
{{ tostring 42 }} → 42
```

#### `tostrings`

Stringify every element of a `[]string` or `[]any`.

```
{{ tostrings (list 1 "x" true) }} → [1 x true]
```

#### `json`

`json prettify v` — marshal to JSON; `prettify=true` indents with 2 spaces.

Input: `.Next = {"a": 1}`

```
{{ .Next | json false }} → {"a":1}
{{ .Next | json true }} → "{\n  \"a\": 1\n}"
```

#### `fromjson`

Parse a JSON object string.

```
{{ index (fromjson `{"a":1}`) "a" }} → 1
```

#### `fromjsonarray`

Parse a JSON array string.

```
{{ index (fromjsonarray `[1,"x",true]`) 1 }} → x
```

### YAML

#### `yaml`

`yaml prettify v` — marshal to YAML; `prettify=true` uses 4-space indent.

```
{{ yaml false (dict "a" 1 "b" "two") }} → a: 1\nb: two
```

#### `fromyaml`

Parse a YAML scalar/document.

```
{{ index (fromyaml "a: 1") "a" }} → 1
```

#### `fromyamlarray`

Parse a YAML array.

```
{{ index (fromyamlarray "- 1\n- 2") 1 }} → 2
```

### Regex

`pattern` uses Go's `regexp/syntax` (RE2). Compile errors abort rendering.

#### `regexmatch`

```
{{ regexmatch "\\d+" "abc 123" }} → true
```

#### `regexfind`

Leftmost match, or `""` if none.

```
{{ regexfind "\\d+" "a 12 b 34" }} → 12
```

#### `regexfindall`

`regexfindall pattern n s` — up to `n` matches; `-1` returns all.

```
{{ regexfindall "\\d+" -1 "1 2 3" }} → [1 2 3]
```

#### `regexreplaceall`

`regexreplaceall pattern repl s` — `$1`-style expansion in `repl`.

```
{{ regexreplaceall "(\\w+)@(\\w+)" "$2/$1" "x@y" }} → y/x
```

#### `regexreplaceallliteral`

Same as `regexreplaceall` but `repl` is taken literally.

```
{{ regexreplaceallliteral "\\d+" "$1" "a 12 b" }} → a $1 b
```

#### `regexsplit`

`regexsplit pattern n s`.

```
{{ regexsplit "," -1 "a,b,c" }} → [a b c]
```

### Encoding

#### `b64enc`

Standard base64 encode.

```
{{ b64enc "hello" }} → aGVsbG8=
```

#### `b64dec`

Standard base64 decode; bad input returns `""`.

```
{{ b64dec "aGVsbG8=" }} → hello
```

#### `b32enc`

Standard base32 encode.

```
{{ b32enc "x" }} → PA======
```

#### `b32dec`

Standard base32 decode.

```
{{ b32dec "PA======" }} → x
```

### Hashes & crypto

#### `sha1`

Hex SHA-1 digest. **Not for security use.**

```
{{ sha1 "abc" }} → a9993e364706816aba3e25717850c26c9cd0d89d
```

#### `sha256`

Hex SHA-256 digest.

```
{{ sha256 "abc" }} → ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad
```

#### `adler32`

Adler-32 checksum as a decimal string.

```
{{ adler32 "abc" }} → 38600999
```

#### `randbytes`

`randbytes n` — `n` bytes from the OS CSPRNG, base64-encoded.

```
{{ randbytes 16 }} → e.g. tQ8FUYjV5p3SfQp42dD2hg==
```

#### `aes_encrypt`

`aes_encrypt key plaintext` — AES-256-CBC. The key string is hashed with SHA-256
to derive the 32-byte AES key. Output is base64(IV || ciphertext).

```
{{ aes_encrypt "secret" "msg" }} → e.g. xZ9b...== (base64)
```

#### `aes_decrypt`

Inverse of `aes_encrypt`.

```
{{ aes_decrypt "secret" (aes_encrypt "secret" "msg") }} → msg
```

### Date & time

`v` is accepted as `time.Time`, `*time.Time`, int/int64 (Unix seconds), or an
RFC3339 string. Bad inputs to non-parsing helpers return `""`.

#### `now`

Current local time.

```
{{ (now).Year }} → e.g. 2026
{{ now | htmldate }} → e.g. 2026-04-28
```

#### `date`

`date layout v` — Go reference layout (`2006-01-02 15:04:05`).

```
{{ date "2006-01-02" "2024-01-02T03:04:05Z" }} → 2024-01-02
```

#### `dateinzone`

`dateinzone layout v zone` — render in IANA zone (UTC fallback for unknown).

```
{{ dateinzone "2006-01-02 15:04" "2024-01-02T00:00:00Z" "America/New_York" }} → 2024-01-01 19:00
```

#### `htmldate`

`htmldate v` — `YYYY-MM-DD`.

```
{{ htmldate "2024-01-02T03:04:05Z" }} → 2024-01-02
```

#### `htmldateinzone`

`htmldateinzone v zone`.

```
{{ htmldateinzone "2024-01-02T00:00:00Z" "UTC" }} → 2024-01-02
```

#### `unixepoch`

Unix-second timestamp as a string.

```
{{ unixepoch "2024-01-02T03:04:05Z" }} → 1704164645
```

#### `ago`

Duration since `v`, as a Go duration string.

```
{{ ago "2024-01-02T00:00:00Z" }} → e.g. 17536h12m4.5s
```

#### `duration`

`duration secs` — Go duration string for a seconds count (int or string).

```
{{ duration 90 }} → 1m30s
```

#### `durationround`

Round duration since `v` to its most significant unit (`s`/`m`/`h`/`d`/`mo`/`y`).

```
{{ durationround "2024-01-02T00:00:00Z" }} → e.g. 2y
```

#### `todate`

Parse a string into a `time.Time`. Errors abort rendering.

```
{{ (todate "2006-01-02" "2024-01-02").Year }} → 2024
```

#### `datemodify`

`datemodify spec v` — apply a Go duration spec (`+1.5h`, `-30m`).

```
{{ (datemodify "1h30m" "2024-01-02T12:00:00Z").Hour }} → 13
```

### Maps / dicts

#### `dict`

Build a `map[string]any` from alternating key/value pairs. Odd-length input
errors.

```
{{ get (dict "a" 1 "b" "two") "b" }} → two
```

#### `get`

`get d key` — value or nil if missing.

```
{{ get (dict "a" 1) "a" }} → 1
```

#### `set`

`set d key value` — mutates `d` and returns it.

```
{{ get (set (dict) "k" "v") "k" }} → v
```

#### `unset`

`unset d key` — mutates `d` and returns it.

```
{{ haskey (unset (dict "a" 1) "a") "a" }} → false
```

#### `haskey`

```
{{ haskey (dict "a" 1) "a" }} → true
```

#### `keys`

Sorted list of map keys.

```
{{ keys (dict "b" 2 "a" 1 "c" 3) }} → [a b c]
```

#### `values`

Map values in key-sorted order.

```
{{ values (dict "b" 2 "a" 1) }} → [1 2]
```

#### `pick`

`pick d k1 k2 …` — subset map containing only the listed keys.

```
{{ keys (pick (dict "a" 1 "b" 2 "c" 3) "a" "c") }} → [a c]
```

#### `pluck`

`pluck key d1 d2 …` — collect `d[key]` from each dict that has it.

```
{{ pluck "a" (dict "a" 1) (dict "a" 2) (dict "b" 3) }} → [1 2]
```

#### `dig`

`dig k1 k2 … fallback d` — nested lookup with a fallback.

Input: `.M = {"a": {"b": "v"}}`

```
{{ dig "a" "b" "fb" .M }} → v
{{ dig "a" "x" "fb" .M }} → fb
```

#### `merge`

Shallow merge into the first map; existing keys win.

```
{{ get (merge (dict "a" 1) (dict "a" 99 "b" 2)) "a" }} → 1
```

#### `mergeoverwrite`

Like `merge` but later maps overwrite.

```
{{ get (mergeoverwrite (dict "a" 1) (dict "a" 99)) "a" }} → 99
```

#### `omit`

`omit d k1 k2 …` — remove keys (case-insensitive).

```
{{ keys (omit (dict "Password" "x" "User" "y" "keep" 1) "password" "USER") }} → [keep]
```

#### `deepcopy`

Recursive copy of nested `map[string]any` / `[]any`.

```
{{ get (deepcopy (dict "a" 1)) "a" }} → 1
```

### Lists

#### `list`

Build an `[]any` from arguments.

```
{{ list 1 2 3 }} → [1 2 3]
```

#### `first`

```
{{ first (list 1 2 3) }} → 1
```

#### `last`

```
{{ last (list 1 2 3) }} → 3
```

#### `rest`

All but the first.

```
{{ rest (list 1 2 3) }} → [2 3]
```

#### `initial`

All but the last.

```
{{ initial (list 1 2 3) }} → [1 2]
```

#### `append`

`append xs v…`

```
{{ append (list 1 2) 3 4 }} → [1 2 3 4]
```

#### `prepend`

`prepend xs v…`

```
{{ prepend (list 2 3) 0 1 }} → [0 1 2 3]
```

#### `concat`

Join multiple lists.

```
{{ concat (list 1 2) (list 3) }} → [1 2 3]
```

#### `reverse`

```
{{ reverse (list 1 2 3) }} → [3 2 1]
```

#### `uniq`

Remove duplicates (first occurrence wins).

```
{{ uniq (list 1 1 2 3 2) }} → [1 2 3]
```

#### `compact`

Remove empty/zero values per `empty`.

```
{{ compact (list 0 1 "" "x" nil "y") }} → [1 x y]
```

#### `without`

`without xs v…` — drop elements equal to any of the omits.

```
{{ without (list 1 2 3 2) 2 }} → [1 3]
```

#### `has`

`has v xs` — does the list contain `v`?

```
{{ has 2 (list 1 2 3) }} → true
```

#### `chunk`

`chunk size xs` — split into groups of `size`.

```
{{ chunk 2 (list 1 2 3 4 5) }} → [[1 2] [3 4] [5]]
```

#### `until`

`until n` — `[0..n-1]`. Negative `n` produces a descending sequence.

```
{{ until 3 }} → [0 1 2]
{{ until -3 }} → [0 -1 -2]
```

#### `untilstep`

`untilstep start end step`.

```
{{ untilstep 1 10 2 }} → [1 3 5 7 9]
{{ untilstep 10 0 -3 }} → [10 7 4 1]
```

#### `seq`

Bash-style range. One arg: `[1..n]`. Two args: `[a..b]`. Three args:
`[start, start+step, …]`.

```
{{ seq 3 }} → [1 2 3]
{{ seq 5 2 }} → [5 4 3 2]
{{ seq 1 2 5 }} → [1 3 5]
```

### Math (integer)

All take any value coercible by `int64`.

#### `add`

Sum of all arguments.

```
{{ add 1 2 3 }} → 6
```

#### `add1`

`v + 1`.

```
{{ add1 5 }} → 6
```

#### `sub`

```
{{ sub 10 3 }} → 7
```

#### `mul`

Product of all arguments.

```
{{ mul 2 3 4 }} → 24
```

#### `div`

Integer division. Division by zero returns `0`.

```
{{ div 10 3 }} → 3
{{ div 10 0 }} → 0
```

#### `mod`

Remainder. Zero divisor returns `0`.

```
{{ mod 10 3 }} → 1
```

#### `max`

```
{{ max 1 5 3 }} → 5
```

#### `min`

```
{{ min 5 1 3 }} → 1
```

### Math (float)

All take any value coercible by `float64`.

#### `addf`

```
{{ addf 1.5 2.5 }} → 4
```

#### `add1f`

`v + 1` as float.

```
{{ add1f 1.5 }} → 2.5
```

#### `subf`

```
{{ subf 5.5 1.5 }} → 4
```

#### `mulf`

```
{{ mulf 2.0 3.0 }} → 6
```

#### `divf`

Float division. Zero divisor returns `0`.

```
{{ divf 7.0 2.0 }} → 3.5
```

#### `maxf`

```
{{ maxf 1.5 2.5 1.0 }} → 2.5
```

#### `minf`

```
{{ minf 2.5 1.5 3.0 }} → 1.5
```

#### `floor`

Largest integer ≤ v, as float.

```
{{ floor 3.9 }} → 3
```

#### `ceil`

Smallest integer ≥ v, as float.

```
{{ ceil 3.1 }} → 4
```

#### `round`

`round v precision` — round to `precision` decimal places.

```
{{ round 3.456 2 }} → 3.46
{{ round 0.5 0 }} → 1
```

### File path

#### `base`

Last element of the path.

```
{{ base "/a/b/c.txt" }} → c.txt
```

#### `dir`

All but the last element.

```
{{ dir "/a/b/c.txt" }} → /a/b
```

#### `clean`

Lexical path cleanup.

```
{{ clean "/a/./b/../c" }} → /a/c
```

#### `ext`

File extension including the leading dot.

```
{{ ext "foo.tar.gz" }} → .gz
```

#### `isabs`

Whether the path is absolute.

```
{{ isabs "/x" }} → true
{{ isabs "rel/x" }} → false
```

### URL

#### `urlparse`

Parse a URL into a map with `scheme`, `host`, `hostname`, `path`, `query`,
`opaque`, `fragment`, `userinfo` keys.

```
{{ (urlparse "https://u:p@example.com:8080/p?x=1#f").host }} → example.com:8080
```

#### `urljoin`

Inverse of `urlparse`. Honours the same map keys.

```
{{ urljoin (dict "scheme" "https" "host" "x" "path" "/p") }} → https://x/p
```

### Identifiers

#### `uuidv4`

Random RFC 4122 v4 UUID.

```
{{ uuidv4 }} → e.g. 7c2b6a4d-9c8e-4f1a-9b2c-3d4e5f6a7b8c
```

### Process environment

#### `env`

`env "VAR"` — `os.Getenv`. Read at render time.

```
{{ env "API_TOKEN" }} → e.g. abc123
```
