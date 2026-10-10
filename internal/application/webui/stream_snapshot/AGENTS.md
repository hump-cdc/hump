# application/webui/stream_snapshot — how this package hangs together

`GET /api/snapshot` (`snapshot.stream`, registered in `internal/controller/webui.go`), an SSE
stream of the snapshot runner's state: the current state on connect, then the state again on every
change. It is the only place the dashboard learns about a run — `start_snapshot` and
`stop_snapshot` answer with empty bodies.

## Layout

| File | Role |
|---|---|
| `usecase.go` | `Subscribe`: registers with the runner, returns the state, the channel and the release func |
| `controller.go` | the SSE handler, `Events()`, `State` → event rendering |
| `dto.go` | the three event payloads and their tables — the wire contract |

## Every event is the whole state

The runner sends `State` **in full** on each change, never a delta, and a subscriber that falls
behind gets the latest state rather than every one (`port/snapshot.Runner.Subscribe`). So a client
needs no history: it can connect mid-run, miss events, or reconnect, and the next event is
complete. Do not add anything incremental to these payloads — it would break exactly that.

The handler sends the state `Subscribe` returned **before** it enters the loop, so a client sees
where things stand without waiting for the next change.

## Three states, three shapes

`event` switches on `State.Status`, and each case renders a different payload, not one struct with
optional fields:

| Event | Table status | Carries |
|---|---|---|
| `idle` | — | `state` only |
| `processing` | `wait` / `processing` / `done` | `where`, `completedRows`, `estimatedRows`, per-sink `count` |
| `done` | `success` / `error` | `error`, `completedRows`, `startedAt` + `finishedAt` |

The table status vocabulary changes between the two: a table that failed mid-run is `done` with an
error while the run is still going, and only becomes `error` once the run finishes — see
`adapter/snapshot/AGENTS.md`, "Statuses". `where` and `estimatedRows` are deliberately absent from
a finished table. An unknown status falls into `default` and is reported as `idle`.

## Changing an event touches two places

`Events()` maps names to payload types for huma, but `cmd/openapi/main.go` separately lists the
three schema names for this route in `flattenSSESchema`, which rewrites the documented response
into a `oneOf` of payloads for the generated TypeScript client. A new or renamed event must be
added there too.

The flattened spec **drops event names**, so the client tells payloads apart by shape. Here that is
the single-value `enum` on each event's `state` field (`idle`, `processing`, `done`) — the reason
`state` is sent at all, when the event name already says it. Keep those tags.

## Leaving

The loop ends three ways, and all of them go through the deferred `release`, which unsubscribes:

- `ctx.Done()` — the client left;
- the channel closed — the runner is shutting down; `adapter/snapshot.close` runs at
  `StageWorkers`, before the web server's `StageServers`, so parked handlers return and
  `server.Shutdown` is not left waiting on them;
- a failed `send`.

With a nil runner `Subscribe` returns an idle state and a **nil channel**, which blocks forever, so
the stream sends one `idle` and then waits on the context alone. A served route never has a nil
runner — `cmd/hump/main.go` builds one whenever the web port is set; the path exists for
`cmd/openapi`, which constructs the use case with nil to register the route, and for tests.

## Tests

`controller_test.go` covers the use case through the handler; there is no `usecase_test.go`.
`fakeRunner` embeds the port, hands out a fixed state and channel, and records the unsubscribe —
both "stops" tests assert on it. `fakeSender` can fail every send or run a hook on send, which is
how `TestHandlerSendsIdleWithoutRunner` cancels the context after the first event. `serve` fails a
test that runs past five seconds. Helpers live below the last test function — see the `go` skill.
