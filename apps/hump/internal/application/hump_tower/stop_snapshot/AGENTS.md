# application/hump_tower/stop_snapshot — how this package hangs together

`POST /api/snapshot/stop` (`snapshot.stop`, `DefaultStatus: 204`; no request body, no response
body). One route, two meanings, decided by the runner's state:

- a run **in progress** is stopped — the tables it did not finish end as `error` `"stopped"`;
- a **finished** run is cleared, taking the runner back to idle.

That is why the dashboard uses the same call for "stop" and for dismissing a finished run.

The use case is a pass-through on purpose. What stopping actually involves — cancelling the run,
releasing the exported snapshot, resuming WAL — is `adapter/snapshot`; see its `AGENTS.md`, "Stop,
restart, shutdown". `Stop` returns once the state has turned; the release happens afterwards on
the run goroutine, and the new state reaches clients over `GET /api/snapshot` (`stream_snapshot`).

## Errors

`ErrNotRunning` (the runner is idle) → 409, a nil runner (`ErrUnavailable`) → 503. Anything else is returned raw, so huma answers 500.
