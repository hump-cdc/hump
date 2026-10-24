# application/hump_tower/get_config — how this package hangs together

`GET /api/config` (`config.get`). It describes the running server to the dashboard: build, the
config it was started with, the source database, and every configured table with the sinks and
targets it is delivered to. Everything comes from what the process was started with — there is no
query, no error path and no state.

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

## `Targets` collapses edges

`Targets` must render exactly like the metrics topology; the rule is in `port/sink/AGENTS.md`.

Unlike the topology, which keeps one connection per configured sink, `tableTargets` **collapses**
sinks that render to the same edge into one entry carrying the union of their operations. `union`
walks `domain.OperationTypes` rather than appending, so the result is always in canonical order
whatever order the config listed them in — `TestConfigCollapsesSinksAndTargets` pins both.

## Things that will bite

- **Lists are never nil.** `Sinks` and `Tables` are allocated even when empty, and the controller's
  `configSinks` / `configTargets` do the same, so the client gets `[]`, never `null`. The
  generated zod schema marks these arrays `.nullable()`, so a `null` would still pass validation;
  the code does not say why it allocates.
- **`Config()` does not guard a nil database**: on the `cmd/openapi` instance it panics on
  `uc.db.GetDbHost()`.
- `RawConfig` is `opts.Config` verbatim — the configuration as it was passed at startup, not a
  re-marshalled one.
- The `operations` enum tag in `dto.go` repeats `domain.OperationTypes` by hand.
