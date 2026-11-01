# application/hump_tower/start_snapshot — how this package hangs together

`POST /api/snapshot/start` (`snapshot.start`, `DefaultStatus: 202`). It asks the snapshot runner to
start a run over the chosen tables and returns at once with no body. Progress is not reported
here — it arrives on `GET /api/snapshot` (`stream_snapshot`).

The use case receives `cfg` by injection and resolves the full batch through
`config.Configuration.ResolveSnapshotTable` into a separate slice without mutating the input.
If any table is denied, it returns `ErrTableNotConfigured` and never calls the runner. On success,
it calls `runner.Start` once with the resolved pairs. Everything a run does — validating filters,
estimating, pausing WAL, reading, finishing — is `adapter/snapshot`; see its `AGENTS.md`.

## Errors

`ErrAlreadyRunning` → 409, `ErrInvalidWhere` and `ErrTableNotConfigured` → 422, a nil runner
(`ErrUnavailable`) → 503. Anything else, `snapshot.ErrClosed` included, is returned raw, so huma
answers 500.

- **Filters are validated by the runner, all of them, before anything starts** — one bad `where`
  fails the whole request and no table is touched. The handler only maps the result.
- **The sentinel is `port/snapshot.ErrInvalidWhere`**, not `port/database.ErrInvalidWhere`. They
  are two different variables with the same message. The runner joins its own onto whatever
  `database.ValidateWhere` returned, and the runner's is the one its port documents — match that.

## Request

`minItems:"1"` on `tables` and `minLength:"1"` on `table` are enforced by huma before the handler.
Schema and table names must match configured pairs exactly, including case. A nonempty request
schema requires that explicitly configured pair. An empty request schema resolves only to a unique
nonempty configured schema for the same table name; duplicate entries for that pair are fine.
Multiple schemas, or any matching configuration entry with an empty schema, make shorthand
unresolvable. There is no `public`, `search_path`, database or partition fallback.
