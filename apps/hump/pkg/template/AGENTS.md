# pkg/template — how this package hangs together

The string type behind every templated sink setting. A `Template` is either a plain string or a
parsed Go `text/template`, decided once when the config is loaded; sinks call `Render` per row
without knowing which. `funcs/` is the function map every parsed template gets.

The user-facing reference (render context, syntax, every function with examples) is
`docs/template.md`. This file covers what the code depends on.

## Layout

| File | Role |
|---|---|
| `template.go` | `Template`, `New`, the YAML (un)marshalling, the static/dynamic dispatch |
| `template_renderable.go` | the parsed variant; wraps execution errors with `ErrRender` |
| `funcs/funcs.go` | `Funcs`, the one `template.FuncMap` |
| `funcs/builtins.go` | the stdlib builtins with logic (`and`, `or`, `not`, `call`, `index`, `slice`, `len`, the comparisons), copied from `text/template` |
| `funcs/unavailable.go` | the `unavailable` hook into `eq`, `ne` and `empty` |
| `funcs/args.go` | `strArg`, nil-tolerant string coercion |
| `funcs/<category>.go` | one file per function group (`case`, `strings`, `dict`, `traceparent`, …) |

## Static or dynamic, and it never fails

`parseTemplate` treats a source as dynamic only if it contains `{{` **and** parses. Everything else
is static and `Render` returns it verbatim, ignoring the data.

The consequence to keep in mind: **a malformed template is not an error anywhere.** `New` always
returns a nil error, `UnmarshalYAML` accepts it, and `{{ .x ` or a call to a misspelled function
goes to the sink as that literal text on every row. `TestUnmarshalYAML_BrokenTemplateFallsBackToStatic`
pins it and `docs/template.md` documents it as behaviour. Config validation that wants to reject a
typo has to do so itself; `IsDynamic` is the only signal this package gives.

Other edges:

- A **nil** `*Template` is usable: `Render` returns `""`, `Source` `""`, `IsDynamic` false. Optional
  config fields rely on that.
- A **zero** `Template{}` is not: its inner variant is nil and every method panics. Build one with
  `New` or by unmarshalling.
- `UnmarshalYAML` accepts only a string scalar. A YAML number or bool in a templated field is
  rejected with `should be string` rather than converted. A block scalar (`|`, `>`) takes its
  decoded value from the inner node: goccy's `GetValue` on a `LiteralNode` is the source text,
  header and indentation included.
- `MarshalYAML` emits the source, so a config round-trips with its templates unrendered. A source
  goccy would not read back from its own output (`"\t"`, `"\n"`) is emitted double-quoted.
- It is `text/template`: nothing is escaped for the destination format. `json`, `quote`, `urlquery`
  and friends are the author's job.

## `ErrRender`

Every execution failure is returned as `ErrRender` wrapping the engine's error. It marks a failure
that comes from the template source or the shape of the data and will not go away on retry; every
sink's `wrapError` classifies it as permanent (`adapter/sink/AGENTS.md`). The retry layer still
retries on any error, so today the marker does not stop retries.

Functions that return `(value, error)` (`required`, `fail`, `fromjson`, `todate`, the regex
family) are how a template raises one deliberately.

## Why the builtins are re-implemented

`Funcs` contains `and`, `or`, `eq`, `index`, `len` and the rest of the stdlib set. The ones in
`builtins.go` are copied from `text/template`; `html`, `js`, `urlquery` and `print*` are the
stdlib functions themselves. A `FuncMap` entry overrides the builtin of the same name, and the
override is needed for one reason: **unavailable values**.

A column the WAL stream did not carry is `domain.UnavailableValue`. It renders as `""`, but it is a
struct, and the stock `eq` refuses to compare a struct with a string. So here:

- `eq` and `ne` substitute `""` for it before comparing (`normalizeUnavailable`).
- `empty`, `coalesce` and `required` treat it as empty.
- `unavailable` is the only function that tells it apart from a real empty string.
- `default` is **not** part of that: it replaces an untyped nil only, so neither an unavailable
  value nor `""` triggers the fallback. Use `coalesce` for those.

`funcs` reaches the value through the one-method `unavailableValue` interface rather than importing
`internal/domain`, which is what keeps `pkg/` free of project imports. The test doubles `undelivered`
and `delivered` exist to prove the check calls the method instead of only looking for it.

The copy went through a mechanical rename, so a few engine error strings read oddly (`tmplIndex out
of range`, `should be 1 tmplOr 2`). They are user-visible in render errors.

## Conventions in `funcs`

- Each function of our own is `tmpl<Name>` in the file of its category, registered in `Funcs` under
  a **lowercase, unseparated** name (`kebabcase`, `hasprefix`, `regexreplaceall`). Exceptions to the
  name rule: `aes_encrypt`, `aes_decrypt`, `trace_id`, `span_id`. Some implementations do not match
  their registered name (`len` → `tmplLength`, `lower` → `tmplLowerCase`); look them up in `Funcs`.
- The subject comes **last**, so it can be piped: `{{ .Next.name | trimprefix "x-" }}`. `get`, `set`,
  `unset`, `haskey`, `pick` and `omit` accept the dict first or last (`dictArgs`); the classic
  dict-first form wins when both ends are maps.
- String helpers that should tolerate a missing field take `any` and go through `strArg`, which
  turns nil into `""`.
- The names follow sprig but the behaviour is not always sprig's: `lower` and `upper` split words
  (`FooBar` → `foo bar`), and conversions and integer math coerce silently (unparseable → `0`,
  division by zero → `0`).
- `htpasswd`, `derivePassword`, `semver` and the TOML helpers are deliberately absent; each would
  add a module dependency.
- `env` and `hostname` are plain `os.Getenv` and `os.Hostname`: a template can read any variable of
  the process. `now`, `uuidv4`, `shuffle` and the `rand*` family differ on every render, so the same
  row retried renders a different message.

`trace_id` and `span_id` validate strictly (lowercase hex, no `ff` version, no all-zero ids) and
disagree on failure: `trace_id` returns its input unchanged, `span_id` returns `""`. The parser the
logger uses is a different, more lenient one; see `pkg/observability/AGENTS.md`.

## Adding a function

1. Implement `tmpl<Name>` in the category file, with a doc comment.
2. Register it in `Funcs`, in the matching block, alphabetically.
3. Add a row to `TestNew_FuncMap_AllCallable` in `template_test.go`, which renders every registered
   name through the real engine.
4. Add the name to the hand-kept list in `TestFuncs_AllRegistered`. That list only checks presence
   and has already drifted: `unavailable`, `trace_id` and `span_id` are missing from it.
5. Cover the edge cases in `funcs/<category>_test.go`.
6. Document it in `docs/template.md`.

## Tests

`funcs/helper_test.go` holds the shared helpers for the whole `funcs` package: `render` and
`mustRender` run a source through `text/template` with `Funcs`, and `V` wraps a value as `.Value`.
The per-category tests go through those rather than calling `tmpl*` functions directly, so a broken
signature surfaces as a parse or execution error. The exception is `unavailable_test.go`, which
also calls the `tmpl*` functions directly on the `undelivered` / `delivered` doubles.

`template_test.go` owns the wiring (static/dynamic dispatch, nil receiver, YAML) and leaves
function edge cases to `funcs`.
