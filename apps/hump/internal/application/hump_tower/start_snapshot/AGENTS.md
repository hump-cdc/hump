# application/hump_tower/start_snapshot — how this package hangs together

`POST /api/snapshot/start` (`snapshot.start`, `DefaultStatus: 202`). It asks the snapshot runner to
start a run over the chosen tables and returns at once with no body. Progress is not reported
here — it arrives on `GET /api/snapshot` (`stream_snapshot`).

The use case is a pass-through on purpose. Everything a run does — validating filters, estimating,
pausing WAL, reading, finishing — is `adapter/snapshot`; see its `AGENTS.md`.

## Errors

`ErrAlreadyRunning` → 409, `ErrInvalidWhere` → 422, a nil runner (`ErrUnavailable`) → 503. Anything else, `snapshot.ErrClosed` included, is returned raw, so
huma answers 500.

- **Filters are validated by the runner, all of them, before anything starts** — one bad `where`
  fails the whole request and no table is touched. The handler only maps the result.
- **The sentinel is `port/snapshot.ErrInvalidWhere`**, not `port/database.ErrInvalidWhere`. They
  are two different variables with the same message. The runner joins its own onto whatever
  `database.ValidateWhere` returned, and the runner's is the one its port documents — match that.

## Request

`minItems:"1"` on `tables` and `minLength:"1"` on `table` are enforced by huma before the handler.
An empty `schema` is passed through as empty and resolved through `search_path` by the database.
