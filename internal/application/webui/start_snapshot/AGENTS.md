# application/webui/start_snapshot — how this package hangs together

`POST /api/snapshot/start` (`snapshot.start`, registered in `internal/controller/webui.go` with
`DefaultStatus: 202`). It asks the snapshot runner to start a run over the chosen tables and
returns at once with no body. Progress is not reported here — it arrives on `GET /api/snapshot`
(`stream_snapshot`).

## Layout

| File | Role |
|---|---|
| `usecase.go` | `Start`: the nil-runner check, then `runner.Start` |
| `controller.go` | the huma handler, the error-to-status mapping, request conversion |
| `dto.go` | `SnapshotStartRequest` — the wire contract |

The use case is a pass-through on purpose. Everything a run does — validating filters, estimating,
pausing WAL, reading, finishing — is `adapter/snapshot`; see its `AGENTS.md`.

## Errors

| Cause | Status |
|---|---|
| `snapshot.ErrAlreadyRunning` | 409 |
| `snapshot.ErrInvalidWhere` | 422 |
| `ErrUnavailable` (nil runner) | 503 |
| anything else, `snapshot.ErrClosed` included | returned raw, so huma answers 500 |

Two things to know about that table:

- **Filters are validated by the runner, all of them, before anything starts** — one bad `where`
  fails the whole request and no table is touched. The handler only maps the result.
- **The sentinel is `port/snapshot.ErrInvalidWhere`**, not `port/database.ErrInvalidWhere`. They
  are two different variables with the same message. The runner joins its own onto whatever
  `database.ValidateWhere` returned, and the runner's is the one its port documents — match that.

`ErrUnavailable` is declared in this package and, separately, in `stop_snapshot`; they are not the
same variable. A served route never has a nil runner — `cmd/hump/main.go` builds one whenever the
web port is set. The nil path exists for `cmd/openapi`, which constructs the use case with nil to
register the route, and for tests.

## Request

`minItems:"1"` on `tables` and `minLength:"1"` on `table` are enforced by huma before the handler.
An empty `schema` is passed through as empty and resolved through `search_path` by the database.
`dto.go` tags are emitted into `web/webui/src/shared/api/openapi.json` by `task openapi`.

## Tests

`controller_test.go` only. `fakeRunner` embeds the port, records the tables it was started with
and answers a fixed error; `request` builds the input. Helpers live below the last test function —
see the `go` skill.
