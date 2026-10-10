# application/webui/stream_updates — how this package hangs together

`GET /api/live` (`live.stream`, registered in `internal/controller/webui.go`), the dashboard's SSE
feed: a `welcome` on connect, then one `operations` event per batching window listing the
deliveries the pipeline made in it. It does a second job that is easy to miss — **a connected
client is what turns the collectors on.**

## Layout

| File | Role |
|---|---|
| `usecase.go` | `Subscribe`: connects both collectors, subscribes to the feed, returns the release func |
| `controller.go` | the SSE handler, `Events()`, batch → event rendering |
| `dto.go` | `WelcomeEvent`, `OperationsEvent`, `OperationEvent` — the wire contract |

## This route is the on-switch

`metrics_collector` and `operations_collector` are idle by default: no scrape loop, no ring, no
cost. They wake on `Connect` and go idle again some time after the last `Disconnect`. **Nothing
else in the web UI calls `Connect`.** `/api/metrics` and `/api/operations` only read, so without an
open `/api/live` they answer zeros and an empty page. See "Idle by default" in
`adapter/metrics_collector/AGENTS.md` and "Nothing at all until a client connects" in
`adapter/operations_collector/AGENTS.md`.

`Subscribe` is one `Connect` per collector per client, and the release func is its exact mirror in
reverse order:

```
metrics connect → operations connect → subscribe
unsubscribe → operations disconnect → metrics disconnect
```

`TestSubscribeAttachesAndReleasesTheClient` pins that sequence. The collectors are refcounted, so a
`Connect` without its `Disconnect` keeps them running for the life of the process — the handler's
`defer release()` is what guarantees the pair on every exit.

`operations` and `stream` are **the same object** in production: `adapter/live_streamer` is a
decorator over the operations collector and `cmd/hump/main.go` passes it as both. They stay two
parameters because they are two ports.

## What an event is, and is not

A batch is everything delivered since the previous one, closed on a timer (500ms, set in
`cmd/hump/main.go`), so a busy pipeline reaches a client as a couple of events a second. An
`OperationEvent` says only **where** a delivery went — table, sink, host, target. No payload, no
timing, no outcome: the feed animates the flow and tells the client something changed, and the
client refetches `/api/metrics` and `/api/operations` for the rest.

A client that cannot keep up **loses batches** rather than slowing the pipeline; the streamer drops
them (`adapter/live_streamer/AGENTS.md`). Nothing here may buffer or retry to compensate.

`welcome` is sent before the loop so a client learns it is connected without waiting for traffic.

## Changing an event touches two places

`Events()` maps names to payload types for huma, but `cmd/openapi/main.go` separately lists the
schema names for this route in `flattenSSESchema`, which rewrites the documented response into a
`oneOf` of payloads for the generated TypeScript client. A new or renamed event must be added there
too. The flattened spec **drops event names**, so payloads have to stay distinguishable by shape —
today an empty object versus one with a required `operations`.

## Leaving

The loop ends on `ctx.Done()` (the client left), on a closed channel (the streamer shutting down —
it closes subscriber channels at `StageWorkers`, before the web server stops, so parked handlers
return), or on a failed `send`. All three run `release`.

Every dependency may be nil, each guarded on its own. A nil stream yields a **nil channel**, which
blocks forever, so the handler sends `welcome` and then waits on the context alone. As wired
today, `cmd/hump/main.go` builds all three whenever the web port is set, so a served route sees
none of them nil — the field comments tying nil to "telemetry disabled" do not describe the current
wiring. The nil paths are real for `cmd/openapi`, which constructs the use case with three nils to
register the route, and for tests.

## Tests

`usecase_test.go` only; the handler loop has none here. `fakeMetrics`, `fakeOperations` and
`fakeStreamer` embed their ports and append to one shared log, which is how the call order is
asserted. Helpers live below the last test function — see the `go` skill.
