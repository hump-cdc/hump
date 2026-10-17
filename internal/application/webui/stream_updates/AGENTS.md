# application/webui/stream_updates — how this package hangs together

`GET /api/live` (`live.stream`, registered in `internal/controller/webui.go`), the dashboard's SSE
feed: a `welcome` and a `snapshot` on connect, then one `operations` event per batching window
listing the deliveries the pipeline made in it, and a `snapshot` whenever the snapshot run changes
state or moves a whole percent. It does a second job that is easy to miss — **a connected
client is what turns the collectors on.**

## Layout

| File | Role |
|---|---|
| `usecase.go` | `Subscribe`: connects both collectors, subscribes to the feed and the snapshot runner, returns the `Feed` and the release func; `progress` |
| `controller.go` | the SSE handler, `Events()`, batch → event and state → event rendering |
| `dto.go` | `WelcomeEvent`, `OperationsEvent`, `OperationEvent`, `SnapshotEvent` — the wire contract |

## This route is the on-switch

`metrics_collector` and `operations_collector` are idle by default: no scrape loop, no ring, no
cost. They wake on `Connect` and go idle again some time after the last `Disconnect`. **Nothing
else in the web UI calls `Connect`.** `/api/metrics` and `/api/operations` only read, so without an
open `/api/live` everything scraped is empty: `/api/metrics` returns no `points`, zero counts and
no `p95` in `tables`, and zero `operations` with no `success` or `p95` in `sinks`, and
`/api/operations` returns an empty page. `connections[].success` and `failure` are the exception.
The pipeline pushes them through `Record` whether or not the collector is active, so they keep
counting. See "Idle by default" in
`adapter/metrics_collector/AGENTS.md` and "Nothing at all until a client connects" in
`adapter/operations_collector/AGENTS.md`.

`Subscribe` is one `Connect` per collector per client, and the release func is its exact mirror in
reverse order:

```
metrics connect → operations connect → subscribe → snapshot subscribe
snapshot unsubscribe → unsubscribe → operations disconnect → metrics disconnect
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

## The snapshot event

`/api/metrics` knows nothing about snapshots, so this stream is how every page learns one is
running. The handler sends the state `Subscribe` returned right after `welcome`, so a client that
connects mid-run needs no other request; `runner.Subscribe` hands out the state and the channel
together, so nothing is lost in between.

`SnapshotEvent` is `state` plus `progress`: rows read across all tables as a whole percent of the
estimate, rounded down (`progress` in `usecase.go`). It is 0 outside `processing` and while the
estimate is unknown — the runner estimates before it reads, and zeroes the estimates when it
finishes. 100 can show briefly, between the last row and the run turning `done`.

The runner throttles per-row progress to five updates a second; everything else (start, estimates,
a table starting or finishing, the end of the run) publishes at once. The handler forwards a state only when its rendered event differs from the last
one sent, which is the whole dedupe. Do not move the
comparison to the raw `State` — that changes on every row.

## Changing an event touches two places

`Events()` maps names to payload types for huma, but `cmd/openapi/main.go` separately lists the
schema names for this route in `flattenSSESchema`, which rewrites the documented response into a
`oneOf` of payloads for the generated TypeScript client. A new or renamed event must be added there
too. The flattened spec **drops event names**, so payloads have to stay distinguishable by shape —
today an empty object, one with a required `operations`, and one with a required `state`. The
client reads `state` to recognise the snapshot event, so no other payload may carry that field.

## Leaving

The loop ends on `ctx.Done()` (the client left), on a closed channel (the streamer shutting down —
it closes subscriber channels at `StageWorkers`, before the web server stops, so parked handlers
return), or on a failed `send`. All three run `release`.

A closed **snapshot** channel is not one of them: the runner shuts down at the same stage, and the
handler just sets its channel to nil and leaves ending the stream to the feed.

Every dependency may be nil, each guarded on its own. A nil stream or runner yields a **nil
channel**, which blocks forever and silences only that case: the other source's events still
arrive. The handler always sends `welcome` and the initial snapshot (`idle` with a nil runner);
with both nil it then waits on the context alone. `cmd/hump/main.go` builds the first three
whenever the web port is set (their field comments: nil when the web UI is disabled), and the
runner whenever the process is not in cleanup mode and the web port is set, so a served route
sees none of them nil. The nil paths are real for `cmd/openapi`, which
constructs the use case with four nils to register the route, and for tests.

## Tests

`usecase_test.go` covers `Subscribe` and `progress`: `fakeMetrics`, `fakeOperations`,
`fakeStreamer` and `fakeRunner` append to one shared log, which is how the call order is asserted.
`controller_test.go` covers the handler: `fakeSender` can fail every send or run a hook on send,
and `serve` fails a test that runs past five seconds. The dedupe test feeds states through an
unbuffered channel, so each is consumed before the next is offered. Helpers live below the last
test function — see the `go` skill.
