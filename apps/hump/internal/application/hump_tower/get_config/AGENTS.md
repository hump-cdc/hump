# application/hump_tower/get_config — how this package hangs together

`GET /api/config` (`config.get`, registered in `internal/controller/hump_tower.go`). It describes
the running server to the dashboard: build, the config it was started with, the source database, and
every configured table with the sinks and targets it is delivered to. Everything comes from what the
process was started with — there is no query, no error path and no state.

## Layout

| File | Role |
|---|---|
| `usecase.go` | `Config()`: builds the description from `options.Options`, `config.Configuration` and the database getters |
| `controller.go` | the huma handler: copies `Config` into `ConfigResponse` field by field |
| `dto.go` | `ConfigResponse` and its parts — the wire contract |

## Three lists, three keys

`Config()` reports the same sinks three ways, and each is deduplicated on a different key:

| Field | Key | Rendered with |
|---|---|---|
| `Config.Sinks` | `(type, host)`, once across **all** tables, first-seen order | `Metadata(sink.Context{})` |
| `Table.Sinks` | `(type, host)`, once per table | `Metadata(sink.Context{})` |
| `Table.Targets` | `(type, host, target)`, once per table | `Metadata(sink.Context{Schema, Table})` |

The first two render against a **zero context**: sink URLs are templates, but in practice they
resolve from the environment, not from the row. One that does depend on the table renders an empty
host there — and a different host in `Targets`, which renders with the table. That asymmetry is
accepted, not a bug to fix by guessing a context.

A `sinkcfg.Config` whose `Sink` is nil is skipped everywhere.

## `Targets` has to match the metrics topology byte for byte

`tableTargets` makes the same `Metadata(sink.Context{Schema, Table})` call as `topology()` in
`cmd/hump/main.go` and `newBinding` in `process_operation`, because the dashboard joins these edges
to `/api/metrics` by `(type, host, target)` as plain strings. Render one side differently and the
flows view shows edges with no counts — see `adapter/metrics_collector/AGENTS.md`, "Topology comes
from outside".

Unlike the topology, which keeps one connection per configured sink, `tableTargets` **collapses**
sinks that render to the same edge into one entry carrying the union of their operations. `union`
walks `domain.OperationTypes` rather than appending, so the result is always in canonical order
whatever order the config listed them in — `TestConfigCollapsesSinksAndTargets` pins both.

## Things that will bite

- **Lists are never nil.** `Sinks` and `Tables` are allocated even when empty, and the controller's
  `configSinks` / `configTargets` do the same, so the client gets `[]`, never `null`. The
  generated zod schema marks these arrays `.nullable()`, so a `null` would still pass validation;
  the code does not say why it allocates.
- **`Config()` is the one use case here that does not guard a nil dependency.** `cmd/openapi`
  builds it with a nil database to register the route and never invokes the handler; calling
  `Config()` on that instance panics on `uc.db.GetDbHost()`.
- `RawConfig` is `opts.Config` verbatim — the configuration as it was passed at startup, not a
  re-marshalled one.
- The `doc`/`enum` tags in `dto.go` are emitted into `apps/hump-tower/src/shared/api/openapi.json`
  by `moon run hump-tower:openapi`. The `operations` enum repeats `domain.OperationTypes` by hand.

## Tests

`usecase_test.go` only; the controller is a field-by-field copy and has none. `fakeDatabase` embeds
the port and answers the five getters with fixed values; `fakeSink` renders a fixed host and a
target that is the table name when it is set to `"{{table}}"`. Helpers live below the last test
function — see the `go` skill.
