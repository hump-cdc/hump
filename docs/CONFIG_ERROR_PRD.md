# PRD — Human-Readable Configuration Errors

## 1. Background

When `hump` is started with an invalid YAML configuration, the user is currently shown raw `yaml.v3` decoder output. Example today:

```
8:28PM ERR unable to parse configuration: yaml: unmarshal errors:
  line 2: field x not found in type struct { URL string "yaml:\"url,omitempty\""; Method string "yaml:\"method,omitempty\""; Body string "yaml:\"body,omitempty\""; Headers map[string]string "yaml:\"headers,omitempty\"" }
  line 2: field x not found in type struct { URL string "yaml:\"url,omitempty\""; Method string "yaml:\"method,omitempty\""; Body string "yaml:\"body,omitempty\""; Headers map[string]string "yaml:\"headers,omitempty\"" }
  line 2: field x not found in type struct { ... }   ← duplicated 4×
  line 2: field x not found in type struct { ... }
```

Three problems make this output unfriendly:

1. **Internal Go struct definitions** leak into the message (`struct { URL string "yaml:..." ... }`). The user doesn't know what `URL`, `Method`, `Body`, `Headers` correspond to in the schema; they only added a key called `x`.
2. **Each error is repeated several times.** The internal `Decode()` helper (`internal/types/config/decoder.go`) re-decodes the same node from several `UnmarshalYAML` paths (the polymorphic `SinkConfig`, then `HTTPSink`/`KafkaSink`, etc.), so a single offending key is reported multiple times.
3. **Errors are emitted as one flat string,** losing slog's structured-field benefits. The user can't grep on `field=x` or `line=2`; downstream JSON-log consumers see only `error="…"`.

The custom unmarshalers in `internal/types/config/` (`config_delivery.go`, `config_retry.go`, `config_sink_http.go`, `config_sink_kafka.go`) already include `line %d:` in their messages — but the **top-level strict-decode path** (KnownFields-on-unknown-key, type-mismatch) does not.

## 2. Goal

Replace the raw `yaml.v3` decoder output with structured, deduplicated, human-readable errors logged via `slog`, of the shape:

```
8:28PM ERR Unknown field 'x' in configuration  field=x line=2 path=sinks[1].http
8:28PM ERR Invalid value 'test' for field 'mode'  field=mode value=test line=10 expected="reliable|async"
```

Concretely:

- One `logger.Error(...)` call per **distinct** configuration problem.
- The log message is a short sentence the user can act on.
- Structured attrs (`field`, `value`, `line`, `column`, `path`, `expected`) carry the machine-readable detail.
- Internal Go type names, `yaml:"..."` struct tags, and the literal phrase `"yaml: unmarshal errors:"` never appear in user output.
- Hump still exits non-zero on a parse failure (no behaviour change there).

## 3. Non-Goals

- Recovering from configuration errors (`hump` still fails fast).
- Validating *semantic* correctness beyond what is already enforced (e.g. "kafka.connection unreachable"). This PRD covers only schema/syntax errors.
- Replacing `gopkg.in/yaml.v3`. The current `Decode()` round-trip limitation (line numbers refer to re-serialised text, see `internal/types/config/decoder.go:13`) is accepted as a known tradeoff for this iteration. A follow-up may switch to a library that preserves the original `*yaml.Node` line on strict-decode (`goccy/go-yaml` is a candidate).
- Localisation. Messages are English-only.
- Producing a machine-readable error file or LSP-style diagnostics output.

## 4. Error Taxonomy

Five user-facing error kinds will be recognised. Every other error falls back to a generic "Configuration error" message with the raw text passed through.

| Kind                 | Trigger                                                                                    | Message template                                            | Attrs                                  |
| -------------------- | ------------------------------------------------------------------------------------------ | ----------------------------------------------------------- | -------------------------------------- |
| `unknown_field`      | `yaml.v3` strict mode reports `field X not found in type ...`                              | `Unknown field 'X' in configuration`                        | `field`, `line`, `path?`               |
| `wrong_type`         | `yaml.v3` reports `cannot unmarshal !!str ... into int` / similar                          | `Field 'X' has wrong type (got Y, want Z)`                  | `field`, `line`, `got`, `want`         |
| `invalid_value`      | A custom `UnmarshalYAML` rejects an enumerated value (e.g. `DeliveryMode`, `FailureMode`)  | `Field 'X' has invalid value 'V'`                           | `field`, `value`, `line`, `expected`   |
| `invalid_duration`   | `time.ParseDuration` failure in `Retry`/`Backoff` unmarshalers                             | `Field 'X' has invalid duration 'V'`                        | `field`, `value`, `line`               |
| `invalid_template`   | `template.ParseTemplate` failure on a templated string (e.g. `http.url`, `kafka.topic`)    | `Field 'X' has invalid template: <parser message>`          | `field`, `line`, `details`             |
| `yaml_syntax`        | `yaml.Unmarshal` itself fails to even build a node tree (bad indentation, unclosed quote)  | `Invalid YAML syntax: <yaml message>`                       | `line?`, `column?`                     |
| _(fallback)_         | Anything else                                                                              | `Configuration error: <raw>`                                | _(none)_                               |

Notes:
- `path?` (e.g. `sinks[1].http`) is best-effort. The current `Decode()` round-trip discards path. **Initial implementation may omit `path` entirely;** it is a nice-to-have, not a requirement.
- `expected` for `invalid_value` is the comma-or-pipe-joined list of allowed enum values when available (it already is for `DeliveryMode`/`FailureMode` — see `config_delivery.go:38`).

## 5. Design

### 5.1 New error type — `internal/types/errs/config_error.go`

```go
package errs

type ConfigErrorKind string

const (
    KindUnknownField     ConfigErrorKind = "unknown_field"
    KindWrongType        ConfigErrorKind = "wrong_type"
    KindInvalidValue     ConfigErrorKind = "invalid_value"
    KindInvalidDuration  ConfigErrorKind = "invalid_duration"
    KindInvalidTemplate  ConfigErrorKind = "invalid_template"
    KindYAMLSyntax       ConfigErrorKind = "yaml_syntax"
    KindGeneric          ConfigErrorKind = "generic"
)

type ConfigError struct {
    Kind     ConfigErrorKind
    Message  string  // already-formatted, human-readable
    Field    string  // optional
    Value    string  // optional, for invalid_value / invalid_duration
    Got      string  // optional, for wrong_type
    Want     string  // optional, for wrong_type
    Expected string  // optional, for invalid_value (enum values)
    Line     int     // 0 if unknown
    Column   int     // 0 if unknown
    Path     string  // optional, dot/bracket path
    Details  string  // optional, sub-parser message
    Cause    error   // wrapped underlying error
}

func (e *ConfigError) Error() string { return e.Message }
func (e *ConfigError) Unwrap() error { return e.Cause }

// Aggregate carries multiple ConfigError values when a single parse pass
// produces more than one distinct problem.
type ConfigErrors struct {
    Errors []*ConfigError
}

func (e *ConfigErrors) Error() string { /* "N configuration errors" */ }
func (e *ConfigErrors) Unwrap() []error { /* convert to []error */ }
```

`ConfigError` is the carrier between the parser and the logger. The logger inspects it and emits one structured `slog` call per entry.

### 5.2 Parser — `internal/config/error_parser.go` (new file)

A pure function:

```go
// ParseYAMLError converts a raw yaml.v3 error (typically *yaml.TypeError or
// an error returned by a custom UnmarshalYAML) into a deduplicated slice of
// ConfigError values. It never returns nil; the worst case is a single
// KindGeneric entry carrying the raw message.
func ParseYAMLError(err error) []*errs.ConfigError
```

Implementation:

1. If `err` is `*yaml.TypeError`, iterate `err.Errors`. Each entry is a string of the form `"line N: <message>"`.
2. For each entry, run it through a small set of regular expressions:
   - `^line (\d+): field (\S+) not found in type ` → `KindUnknownField`, capture `Line`, `Field` (the struct-tag suffix is discarded).
   - `^line (\d+): cannot unmarshal !!(\w+) into (\S+)` → `KindWrongType`, capture `Line`, `Got`, `Want`.
   - `^line (\d+): invalid delivery mode "(.+?)" \(want (.+?)\):` → `KindInvalidValue`, capture `Line`, `Value`, `Expected`. (One regex per custom enum unmarshaler, kept beside the unmarshaler that emits the message — see §5.4 for a more durable alternative.)
3. Deduplicate by `(Kind, Field, Line, Value)`. The current "4× repeat" is exactly this case.
4. Anything that matches no regex becomes a single `KindGeneric` entry with the raw line text.
5. If `err` is not a `*yaml.TypeError`, attempt the same regexes against `err.Error()`; otherwise fall back to `KindGeneric` with `err.Error()`.

Why regex on strings: `yaml.v3` does not expose typed sub-errors. The custom unmarshalers in this repo already produce flat strings via `fmt.Errorf`. A typed-error-from-source rewrite is out of scope (see §3).

### 5.3 Custom unmarshalers — `internal/types/config/*.go` (modified)

To avoid brittle regex coupling, the custom unmarshalers should return `*errs.ConfigError` directly instead of `fmt.Errorf("line %d: …", …)`. Files affected:

- `config_delivery.go:38` — `DeliveryMode.UnmarshalYAML` → `KindInvalidValue` (`field="delivery.mode"`, `Value`, `Expected="reliable|async"`, `Line=node.Line`).
- `config_retry.go:74,111,117,129` — backoff/retry duration parses → `KindInvalidDuration`.
- `config_sink_http.go:73`, `config_sink_kafka.go:73` — template parse failures → `KindInvalidTemplate`.
- Any other call site that currently uses `fmt.Errorf("line %d: …", node.Line, …)`.

`yaml.v3` will still wrap a returned non-`yaml.TypeError` error inside its own `TypeError.Errors[]` as a string — the message is preserved verbatim but the typed information is lost on round-trip. To avoid that loss, the call site for the **top-level decode** in `internal/config/config.go:33` (`config.Decode(expanded, &cfg)`) should:

- After `dec.Decode(out)`, inspect the returned error.
- If it is `*yaml.TypeError`, hand the whole thing to `ParseYAMLError` (per §5.2).
- If it is `*errs.ConfigError` or `*errs.ConfigErrors`, **return it directly** without wrapping in `fmt.Errorf`. This requires either:
  - Calling custom unmarshalers manually outside of `yaml.NewDecoder`, OR
  - Accepting the round-trip loss and relying on the regex parser in §5.2 to reconstruct typed errors from the message strings.

**Recommended**: accept the round-trip loss for v1. Keep §5.2's regexes as the single source of error translation. Removing the round-trip is tracked as a follow-up (would need swapping `yaml.v3` strict-decode for an on-`*yaml.Node` walker).

### 5.4 Pipeline — `internal/pipeline/pipeline.go` (modified)

`pipeline.Start` currently does (lines 28–60):

```go
cfg, err := cfg.Parse(`…`)
if err != nil {
    return fmt.Errorf("unable to parse configuration: %w", err)
}
```

It will instead branch on `*errs.ConfigErrors` / `*errs.ConfigError`:

```go
cfg, err := cfg.Parse(`…`)
if err != nil {
    var multi *errs.ConfigErrors
    var single *errs.ConfigError
    switch {
    case errors.As(err, &multi):
        for _, ce := range multi.Errors {
            logConfigError(ce)
        }
    case errors.As(err, &single):
        logConfigError(single)
    default:
        logger.Error("Failed to parse configuration", slog.Any("error", err))
    }
    return errs.ErrConfigurationParse // sentinel — main.go just exits 1
}
```

`logConfigError(ce *errs.ConfigError)` builds the `slog` call by appending only the non-zero fields as attrs:

```go
attrs := []any{}
if ce.Field != ""    { attrs = append(attrs, slog.String("field", ce.Field)) }
if ce.Value != ""    { attrs = append(attrs, slog.String("value", ce.Value)) }
if ce.Got != ""      { attrs = append(attrs, slog.String("got", ce.Got)) }
if ce.Want != ""     { attrs = append(attrs, slog.String("want", ce.Want)) }
if ce.Expected != "" { attrs = append(attrs, slog.String("expected", ce.Expected)) }
if ce.Line > 0       { attrs = append(attrs, slog.Int("line", ce.Line)) }
if ce.Column > 0     { attrs = append(attrs, slog.Int("column", ce.Column)) }
if ce.Path != ""     { attrs = append(attrs, slog.String("path", ce.Path)) }
if ce.Details != ""  { attrs = append(attrs, slog.String("details", ce.Details)) }
logger.Error(ce.Message, attrs...)
```

### 5.5 `main.go`

No changes needed if `pipeline.Start` returns `errs.ErrConfigurationParse` after logging. `main.go` (`cmd/hump/main.go:51`) currently calls `logger.Error(err.Error(), …)` — for the sentinel that produces a single, terse line. Acceptable.

Alternative: have `pipeline.Start` return `nil` after logging the config errors and signal exit some other way. **Rejected** — would break the existing `app.Run` / `r.Wait()` contract.

## 6. User-Visible Output — Before / After

**Before** (current):

```
8:28PM ERR unable to parse configuration: yaml: unmarshal errors:
  line 2: field x not found in type struct { URL string "yaml:\"url,omitempty\""; Method string "yaml:\"method,omitempty\""; Body string "yaml:\"body,omitempty\""; Headers map[string]string "yaml:\"headers,omitempty\"" }
  line 2: field x not found in type struct { … } error="unable to parse configuration: yaml: unmarshal errors:\n  line 2: field x not found in type struct { …"
  … (×4)
```

**After** (target):

```
8:28PM ERR Unknown field 'x' in configuration  field=x line=2
8:28PM ERR Failed to parse configuration  area=pipeline
```

Multi-error example (two distinct problems):

```
8:28PM ERR Unknown field 'foo' in configuration  field=foo line=4
8:28PM ERR Field 'delivery.mode' has invalid value 'test'  field=delivery.mode value=test line=10 expected="reliable|async"
8:28PM ERR Failed to parse configuration  area=pipeline
```

## 7. Acceptance Criteria

1. Triggering the original failing config (unknown field `x` on `http:` sink) produces **exactly one** `Unknown field 'x' in configuration` line plus the trailing `Failed to parse configuration` sentinel — no duplicates, no struct dump.
2. A YAML with `delivery.mode: test` produces `Field 'delivery.mode' has invalid value 'test'` with `value=test` and `expected="reliable|async"` as structured attrs.
3. A YAML with a type mismatch (e.g. `maxConcurrent: "many"`) produces `Field 'maxConcurrent' has wrong type (got string, want int)`.
4. A YAML with a malformed Go template (e.g. `partitionKey: "{{ .Next.id"`) — currently silently treated as static per `docs/template.md` — is **unchanged**. (Template parse errors here are silent by design; only fields that go through `template.ParseTemplate` at unmarshal time surface as `invalid_template`.)
5. A YAML with broken syntax (unclosed quote on a single line) produces a single `Invalid YAML syntax` line with `line`/`column` attrs if `yaml.v3` provides them.
6. Process still exits non-zero on any parse failure.
7. No regression in successful-config startup logging.
8. `cmd/hump/main.go` is unchanged or has only trivial edits.
9. Unit tests in `internal/config/error_parser_test.go` cover each `Kind` and the deduplication case (the 4× repeat).

## 8. Verification Plan

End-to-end:

1. Edit the embedded YAML in `internal/pipeline/pipeline.go:28` to inject each failure mode (or — better — temporarily change `Parse` to read from `config.yaml` and edit that). For each, run `go run ./cmd/hump` and confirm the output matches §6.
2. Confirm structured fields show up when the tint handler is bypassed (run with output piped to a non-TTY: `go run ./cmd/hump 2>&1 | cat`).

Unit:

```
go test ./internal/config/...
go test ./internal/types/config/...
```

`error_parser_test.go` table-driven cases:
- `*yaml.TypeError` with 1, 2, and 4 duplicate "field X not found" entries → deduplicates correctly.
- `*yaml.TypeError` with mixed kinds (unknown + wrong-type).
- Direct `*errs.ConfigError` from a custom unmarshaler passes through unchanged.
- Unrecognised string → `KindGeneric`.
- `nil` error → empty slice.

## 9. Critical Files

| Path                                                      | Change   |
| --------------------------------------------------------- | -------- |
| `internal/types/errs/config_error.go`                     | new      |
| `internal/types/errs/errors.go`                           | no change (sentinel re-used) |
| `internal/config/error_parser.go`                         | new      |
| `internal/config/error_parser_test.go`                    | new      |
| `internal/config/config.go`                               | wrap initial `yaml.Unmarshal` and `Decode()` errors via `ParseYAMLError` |
| `internal/types/config/config_delivery.go`                | return `*ConfigError` instead of `fmt.Errorf` |
| `internal/types/config/config_retry.go`                   | same |
| `internal/types/config/config_sink_http.go`               | same |
| `internal/types/config/config_sink_kafka.go`              | same |
| `internal/pipeline/pipeline.go`                           | branch on `*ConfigErrors` / `*ConfigError`, emit structured logs |
| `cmd/hump/main.go`                                        | no change |
