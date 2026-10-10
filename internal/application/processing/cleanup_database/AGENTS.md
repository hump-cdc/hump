# application/processing/cleanup_database — how this package hangs together

The use case behind `--cleanup`: drop the publication and the replication slot hump created, so a
database it no longer reads from stops retaining WAL for it.

## Nothing calls it

**No package imports this one.** The `opts.Cleanup` branch of `cmd/hump/main.go` calls
`db.Cleanup(ctx)` on the database port directly and returns `app.ErrShutdown` to stop the runner.
`UseCase` is therefore dead code today, and the `"Cleanup complete"` log line it would add is never
written. Before changing behaviour here, check which of the two is meant to be the entrypoint —
editing this file alone changes nothing at runtime.

## What it is

| File | Role |
|---|---|
| `usecase.go` | `UseCase.Cleanup`: `db.Cleanup`, then one info line with publication and slot names |
| `logger.go` | `log.Scope("application/cleanup_database")` |

There is no controller, unlike `process_operation` and the `webui` use cases, and no shutdown hook:
the call is one-shot and holds nothing. An error from `db.Cleanup` is returned as is, without the
log line. What cleanup actually drops, and in which order, is the adapter's business — see
`adapter/postgres/postgres_cleanup.go`.

## Naming

The directory is `cleanup_database` but the package is `cleanupdatabase`, so an import needs no
alias to be a valid identifier. Its sibling does the opposite (`package process_operation`). The
logger scope follows the directory name in both.

## Tests

None.
