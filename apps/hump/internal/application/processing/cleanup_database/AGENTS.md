# application/processing/cleanup_database — how this package hangs together

The use case behind `--cleanup`: drop the publication and the replication slot hump created, so a
database it no longer reads from stops retaining WAL for it.

## Who calls it

Only `cmd/hump/main.go`. Its `opts.Cleanup` branch runs `NewUseCase(db).Cleanup(ctx)` as the only
worker and returns `app.ErrShutdown` on success, which stops the runner with a clean exit.

## What it is

`UseCase.Cleanup` calls `db.Cleanup`, then logs one info line with the publication and slot names.
An error from `db.Cleanup` is returned as is, without the log line. There is no shutdown hook: the
call is one-shot and holds nothing. What cleanup actually drops, and in which order, is the
adapter's business — see `adapter/postgres/postgres_cleanup.go`.

## Naming

The package is `cleanupdatabase`, but the logger scope follows the directory name
(`application/cleanup_database`), as in `process_operation`.
