# application/webui/stop_snapshot — how this package hangs together

`POST /api/snapshot/stop` (`snapshot.stop`, registered in `internal/controller/webui.go` with
`DefaultStatus: 204`; no request body, no response body). One route, two meanings, decided by the
runner's state:

- a run **in progress** is stopped — the tables it did not finish end as `error` `"stopped"`;
- a **finished** run is cleared, taking the runner back to idle.

That is why the dashboard uses the same call for "stop" and for dismissing a finished run.

## Layout

| File | Role |
|---|---|
| `usecase.go` | `Stop`: the nil-runner check, then `runner.Stop` |
| `controller.go` | the huma handler and the error-to-status mapping |

There is no `dto.go`: the route has neither input nor output.

The use case is a pass-through on purpose. What stopping actually involves — cancelling the run,
releasing the exported snapshot, resuming WAL — is `adapter/snapshot`; see its `AGENTS.md`, "Stop,
restart, shutdown". `Stop` returns once the state has turned; the release happens afterwards on
the run goroutine, and the new state reaches clients over `GET /api/snapshot` (`stream_snapshot`).

## Errors

| Cause | Status |
|---|---|
| `snapshot.ErrNotRunning` (the runner is idle) | 409 |
| `ErrUnavailable` (nil runner) | 503 |
| anything else | returned raw, so huma answers 500 |

`ErrUnavailable` is declared in this package and, separately, in `start_snapshot`; they are not the
same variable. A served route never has a nil runner — `cmd/hump/main.go` builds one whenever the
web port is set. The nil path exists for `cmd/openapi`, which constructs the use case with nil to
register the route, and for tests.

## Tests

`controller_test.go` only. `fakeRunner` embeds the port, records that `Stop` was called and answers
a fixed error. Helpers live below the last test function — see the `go` skill.
