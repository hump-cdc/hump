# application/processing/cleanup_database — how this package hangs together

The use case behind `--cleanup`: drop the publication and the replication slot hump created, so a
database it no longer reads from stops retaining WAL for it.

## Who calls it

Only `cmd/hump/main.go`. Its `opts.Cleanup` branch runs `NewUseCase(db).Cleanup(ctx)` as the only
worker and returns `app.ErrShutdown` on success, which stops the runner with a clean exit.

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

`TestCleanupCleansTheDatabase` checks that `Cleanup` calls `db.Cleanup` once and returns nil;
`TestCleanupReturnsTheDatabaseError` checks that the port's error comes back as is. The
`fakeDatabase` embeds `database.Database`, so any port method it does not override panics.
