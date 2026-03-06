# Config Parser — Product Requirements Document

## 1. Overview

A YAML configuration parser/validator for `hump`. Ingests a YAML string, validates structure and semantics per the rules below, resolves sink presets, and returns a normalized, in-memory `Config` object ready for runtime consumption.

Single public entry point:

```go
cfg, err := config.Parse(raw string) (*Config, error)
```

Package location: `internal/config`.

## 2. Goals / Non-Goals

### Goals
- Strict, fail-loud parsing of the YAML format described in `CONFIG.md`.
- Resolve `preset` references and per-field overrides into flat sink objects.
- Compile templates at parse time so the runtime receives ready-to-execute templates.
- Produce IDE-friendly errors with YAML path + line/column + message.
- Deduplicate sinks so identical configurations are represented by a single `Sink` instance.

### Non-Goals (v1)
- Kafka auth / TLS / SASL.
- Loading templates from external files.
- Top-level metadata (`version`, `defaults`, etc.).
- File-loading convenience (`ParseFile`) — caller reads the file.
- A separate `Validate(*Config)` entry point.
- Fuzz tests, benchmarks.

## 3. Public API

```go
package config

// Parse parses the given YAML source and returns a validated, resolved Config.
// Returns a *ValidationError (possibly containing multiple issues) on failure.
func Parse(src string) (*Config, error)

type Config struct {
    Sinks  []Sink   // deduplicated set of sinks actually referenced by tables
    Tables []*Table // flat list; multi-table groups expanded; duplicates merged
}

type Table struct {
    Schema string
    Name   string   // Postgres table name (YAML key is `table` in single form, `name` in multi form)
    Sinks  []Sink   // interface values shared with Config.Sinks (same *KafkaSink / *HTTPSink instances)
}

// SinkType enumerates supported sink kinds.
type SinkType string

const (
    SinkTypeKafka SinkType = "kafka"
    SinkTypeHTTP  SinkType = "http"
)

// Sink is one of: *KafkaSink, *HTTPSink.
type Sink interface {
    Type() SinkType
}

type KafkaSink struct {
    Name         string             // preset name if derived from a preset, else ""
    Connection   string
    Topic        *template.Template
    PartitionKey *template.Template // optional; nil if not set
    Payload      *template.Template
}

type HTTPSink struct {
    Name    string
    URL     *template.Template
    Method  *template.Template             // rendered value normalized to uppercase at runtime
    Headers map[string]*template.Template  // optional
    Body    *template.Template
}

type ValidationError struct {
    Issues []Issue
}
func (v *ValidationError) Error() string

type Issue struct {
    Path    string // e.g. "sinks[0].connection" or "tables[1].sinks[0].preset"
    Line    int    // 1-based YAML source line (0 if unknown)
    Column  int    // 1-based YAML source column (0 if unknown)
    Message string
}
```

**Error strategy:** validation is exhaustive — the parser collects every issue it can before returning, rather than bailing on the first error.

## 4. YAML Schema

### 4.1 Top-level
Only two keys allowed:
- `sinks` (optional): list of sink presets.
- `tables` (required): list of table configurations; must be non-empty.

Unknown top-level keys → `ValidationError`.

### 4.2 `sinks` (presets)

Each entry requires:
- `name` (string, unique across `sinks`).
- `type` (`kafka` | `http`).

Type-specific required fields:

| type  | required                                    | optional       |
|-------|---------------------------------------------|----------------|
| kafka | `connection`, `topic`, `payload`            | `partitionKey` |
| http  | `url`, `method`, `body`                     | `headers`      |

Unknown fields for the resolved `type` → error (e.g. `url` inside a `kafka` preset).

### 4.3 `tables`

Each entry is **one of** (mutually exclusive):

**Single-table form:**
```yaml
- schema: public
  table: users
  sinks: [...]
```

**Multi-table form:**
```yaml
- tables:
    - schema: public
      name: users
    - schema: public
      name: orders
  sinks: [...]
```

Rules:
- Single-table uses keys `schema` + `table`.
- Multi-table list items use keys `schema` + `name`.
- Having both `schema`/`table` and nested `tables:` in the same entry → error.
- Having neither → error.
- Sibling `sinks` in a multi-table entry applies to **every** nested table (fan-out).
- `schema` defaults to `"public"` when omitted.
- `schema` and `name`/`table` must match Postgres identifier rules: `^[A-Za-z_][A-Za-z0-9_]{0,62}$`.
- Each table must have ≥ 1 sink after resolution; empty `sinks:` → error.

### 4.4 Per-table `sinks` entries

An entry in a table's `sinks` list is either:
1. A reference: `{ preset: <name> [, overrides...] }`.
2. A full inline sink: same shape as a `sinks` preset **minus** `name`.

Rules:
- `name` is a field **only** valid inside top-level `sinks:` (preset definitions). Using `name` inside a table's sink entry → error.
- `preset` must reference an existing top-level sink by name; unknown preset → error (all collected).
- Everything else from the preset can be overridden except `type`. Trying to set `type` on a preset-referencing entry → error.
- Fields irrelevant to the resolved `type` → error (strict; treated as "unknown field").
- After applying overrides, all fields required for the sink type must be populated.

### 4.5 Strict YAML

The parser uses `gopkg.in/yaml.v3` with `yaml.Decoder.KnownFields(true)`. Any unrecognized key at any level produces an error with its YAML line/column.

### 4.6 Environment interpolation

Before YAML decoding, string scalar values support shell-style substitution:
- `${VAR}` → value of env var `VAR` (empty if unset).
- `${VAR:-default}` → value of env var `VAR`, or `default` if unset/empty.

Applied to the raw text before YAML parse. Escaping: `$$` → literal `$`.

## 5. Template Handling

Fields `topic`, `partitionKey`, `payload` (kafka) and `url`, `method`, `body`, each `headers` value (http) are Go `text/template` strings.

- Templates are parsed and compiled at `config.Parse` time using `template.New(path).Option("missingkey=error").Parse(...)`.
- Syntax errors → validation issue with path + YAML line/column.
- **Variable validation:** each compiled template is walked; any top-level action node referencing a field not in the known schema → validation issue.

Template data schema (the only permitted top-level names):

| Name       | Type              | Meaning                                        |
|------------|-------------------|------------------------------------------------|
| `.schema`  | string            | source Postgres schema                         |
| `.table`   | string            | source Postgres table                          |
| `.values`  | map[string]any    | current row's column → value map; `.values.id` |

Lowercase on purpose (matches YAML style). `{{ .id }}` (flat column access) is **not** supported — must use `{{ .values.id }}`.

The compiled `*template.Template` is stored on the resolved Sink; runtime just executes it.

## 6. Resolution Semantics

### 6.1 Sink resolution
For each entry in a table's `sinks:`:
1. If `preset: X` is present: start from a deep copy of preset `X`.
2. Apply each overridable field present in the entry on top of the copy.
3. If no `preset`, build a standalone sink from the inline fields (requires `type` plus all required fields for that type).
4. Resolve `name`: set to the preset name when the entry is an unmodified reference (no overrides); otherwise `""`.

### 6.2 Sink deduplication
After resolving every table's sinks, the parser interns sinks into `Config.Sinks`:
- Two resolved sinks are **equal** iff all their typed fields are equal (templates compared by their original source text; headers by key+source-text map equality).
- `Config.Sinks` contains one `*Sink` per equivalence class.
- `Table.Sinks` are pointers into `Config.Sinks`.

The goal: minimal `Sinks` slice, but distinct configurations stay distinct.

### 6.3 Table deduplication
If the same `(schema, name)` pair appears across multiple top-level entries:
- The entries are merged into a single `Table`.
- `Sinks` are combined (union via the deduplication in §6.2).

## 7. Validation Rules (summary)

The parser collects **all** of the following as `Issue`s in a single `ValidationError`:

1. YAML syntax errors (line/column from `yaml.v3`).
2. Unknown fields anywhere.
3. `tables` empty or missing.
4. Table entry with both/neither of `{schema+table}` / `tables:`.
5. `schema`, `table`, `name` fail Postgres identifier regex.
6. Table with zero resolved sinks.
7. `sinks[].name` missing, empty, or duplicated.
8. `sinks[].type` missing or not in `{kafka, http}`.
9. Type-required fields missing on a preset or on a fully-resolved sink.
10. Table sink entry with `name` field.
11. Table sink entry with `preset` referencing an unknown sink.
12. Table sink entry overriding `type`.
13. Table sink entry carrying a field invalid for the resolved type.
14. Template compile errors in any of: `topic`, `partitionKey`, `payload`, `url`, `method`, `body`, or any header value.
15. Template referencing a variable outside `{schema, table, values}`.

## 8. Error Format

```go
type Issue struct {
    Path    string
    Line    int
    Column  int
    Message string
}
```

- `Path` uses dotted/bracketed JSON-path-like notation: `sinks[0].payload`, `tables[2].sinks[1].preset`, `tables[0].tables[1].schema`.
- `Line`/`Column` come from `yaml.Node` positions (post-interpolation). When a logical issue has no source position (e.g. a cross-reference failure), they are `0`.
- `Message` is lowercase English, no trailing period (Go convention), e.g. `missing required field "topic"`, `unknown preset "k9"`, `template references unknown variable ".Foo"`.
- `ValidationError.Error()` joins issues with newlines; each line `"<path>:<line>:<col>: <message>"` (columns omitted when zero).

## 9. Implementation Notes

- **Parse pipeline:**
  1. Env interpolation on raw string.
  2. `yaml.Unmarshal` into an intermediate `rawConfig` struct with `yaml.Node` fields preserved for position reporting.
  3. `KnownFields(true)` gates unknown keys.
  4. Build presets map (`name → *Sink`), check uniqueness.
  5. Expand multi-table entries into a flat `[]*Table`.
  6. Resolve each table's sinks (preset lookup + overrides + type validation + template compile + variable walk).
  7. Merge duplicate `(schema, name)` tables.
  8. Intern/dedupe sinks.
- **Template variable walk:** traverse `parse.Tree.Root` nodes; for each `*parse.ActionNode` pick out the first identifier of pipe commands; reject names not in the allow-list. Permit field chains only off known roots (e.g. `.values.anything` is OK; `.foo` is not).
- **YAML library:** `gopkg.in/yaml.v3` exclusively. No extra deps beyond stdlib `text/template` and YAML.

## 10. Testing

**In scope:**
- Table-driven unit tests, one table per validation rule in §7 (happy path + negative).
- Golden YAML fixtures under `internal/config/testdata/` — each fixture paired with either an expected serialized `Config` or an expected `ValidationError` Issue list.

**Out of scope for v1:** fuzz tests, benchmarks.

## 11. Open Assumptions

- [assumption] `schema` default `"public"` applies only when the key is omitted entirely; an explicit empty string is still an error.
- [assumption] Postgres identifier validation uses syntax-only regex; reserved-word detection is not performed since Postgres supports quoted identifiers.
- [assumption] Env interpolation operates on raw YAML source text; a `$` inside a block scalar is interpolated unless escaped as `$$`.
- [assumption] HTTP `method` is a template; its rendered value is uppercased at runtime, and method-token validation happens at that point rather than at parse time.
- [assumption] Sink equality for dedup uses template **source strings**, not compiled AST identity.
- [assumption] When the same `(schema, name)` appears in multiple entries, sink union preserves first-occurrence order after interning.
