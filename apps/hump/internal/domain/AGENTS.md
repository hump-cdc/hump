# domain — how this package hangs together

The vocabulary every other layer shares: what a change event is made of. It declares no error
values either, and the one set of methods is `UnavailableValue`'s marker and serializers (see below). The production files
have no imports at all, and that is the invariant to keep: a type that needs a project or
third-party import belongs in a port or an adapter. (`unavailable_test.go` imports `pkg/template`
and `goccy/go-yaml`, which is fine — tests only.)

One concept per file, named after it. `OperationType`, `DbName`, `TableSchema`, `TableName`,
`Headers` and `DbRecord` are the field types of `port/sink.Context`, which is what a template
sees as `.Operation`, `.Database`, `.Schema`, `.Table`, `.Headers`, `.Prev`, `.Next`.
`DeliveryStatus`, `ReplicaIdentity` and `TableIdentifier` are not part of it.

## The string values are wire formats

Every constant here is a plain `string` type, cast with `string(x)` / `domain.T(s)` at the
edges and never translated. So the literal is the contract, in several places at once:

- **`OperationType`** — the `operations:` list in sink YAML, the `operation` Prometheus label,
  the web API JSON, and `.Operation` in user templates.
- **`DeliveryStatus`** — the `status` Prometheus label, which `metrics_collector/scrape.go`
  casts straight back into a `DeliveryStatus`, and the operations-log filter.
- **`ReplicaIdentity`** — the `replica:` key of a table in YAML.

Renaming a value is a breaking change to config files, dashboards and templates, not a
refactor. The allowed sets are also repeated as struct tags, which nothing derives from the
constants: `oneof=insert update delete truncate` in `config/sink/config.go`,
`enum:"insert,update,delete,truncate"` in `hump_tower/get_config/dto.go` and
`hump_tower/get_operations/dto.go`, `enum:"success,failure"` (no `retry`) in
`hump_tower/get_operations/dto.go`, and `oneof=full unmanaged` in `config/config.go`. Adding a
constant means editing those too.

`OperationTypes` is both the default (a sink with no `operations:` gets a clone of it) and the
canonical order (`get_config` sorts a union of operations by it). A new operation type has to
be appended there or it is neither delivered by default nor listed.

`DeliveryStatusRetry` is an attempt, not an outcome: it is counted on its own metric series and
set on individual attempts in the operations log, but a delivery as a whole ends as `success`
or `failure` only.

## `DbRecord` has three kinds of "nothing"

- a **nil record** — there is no such tuple: `Prev` of an insert or of a snapshot row, both
  sides of a truncate;
- a **`nil` value** — the column is SQL `NULL`;
- **`Unavailable`** — the column exists but the WAL stream did not carry it (an unchanged TOAST
  value, or a non-key column of a key-only old tuple). Only `adapter/postgres` `decodeRow`
  produces it, and only for tables whose replica identity is not FULL — see
  `adapter/postgres/AGENTS.md`.

Never collapse `Unavailable` into `nil`: that would report a fabricated `NULL` to a sink.

## `Unavailable` has an explicit wire marker

`UnavailableValue` is an empty struct with value receivers, so `v == domain.Unavailable` works
and so does a `DbRecord` holding it by value. `String`, `MarshalJSON` and `MarshalYAML` all
yield the string `__hump_unavailable_value`; every serialization path that reads records must
preserve that marker rather than emit `{}`, `nil` or `""`. Real text can match the marker, so
only the in-memory sentinel is unambiguous.

`pkg/template` must not import this package, so the template funcs recognise the value by
duck typing: `pkg/template/funcs/unavailable.go` declares its own `IsUnavailable() bool`
interface. That method is the whole link between the two — do not rename or remove it.

`unavailable_test.go` is the lock on user-visible behaviour, through real templates: the value
is empty for `empty`, `coalesce` and `required`, equal to `""` for `eq`/`ne`, and detectable by
`unavailable`. `default` does **not** replace it (`{{ .Next.blob | default "n/a" }}`
renders `__hump_unavailable_value`). `docs/template.md` describes the same behaviour; change both
together.

## Tables

`TableIdentifier` is used as a map key throughout (`adapter/postgres` registry,
`adapter/snapshot`), so it has to stay comparable — no slices, maps or pointers in it. It has
no `String()`: each consumer formats or quotes the pair for its own purpose, because a metric
label, a log line and a SQL identifier need different things.
