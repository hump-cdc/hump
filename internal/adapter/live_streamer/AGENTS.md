# adapter/live_streamer — how this package hangs together

The dashboard's live feed. It is a **decorator** over `port/operations_collector.Collector`: every
method delegates to the collector it wraps, and `Record` additionally adds the delivery to an open
window that closes on a timer. The pipeline keeps writing to one collector; the SSE handler reads
the same deliveries off the wrapper.

Nothing in `process_operation` knows this package exists. That is the point of the shape: `binding`
holds the `OperationLog` interface it always held, and `cmd/hump/main.go` hands it the wrapper
instead of the collector.

## Layout

| File | Role |
|---|---|
| `live_streamer.go` | the `Streamer`: options, the six delegating methods, the window, the fan-out |
| `logger.go` | the package logger scope |

## The window

`Record` appends to `pending` and arms a `time.AfterFunc` **only when the window was empty**, so a
burst costs one timer per interval rather than one per delivery. `flush` hands the accumulated
slice to every subscriber and sets `pending = nil`; the next delivery allocates a fresh slice sized
from `hint`, the length of the last window. One allocation per window — two a second at the default
interval — against ten-ish growth reallocations a virgin `append` chain would do.

**There is no permanent goroutine.** No ticker, no `run` loop, nothing alive between windows. That
is deliberate and `TestStreamerHasNoBackgroundGoroutines` pins it with goleak, the same way
`operations_collector` does. The dead `FanOutStreamer` that used to live in `internal/port/live_stream`
had an unbuffered `in` channel plus a permanent `run` goroutine, which is exactly what this replaces.

**Do not double-buffer `pending`.** A second buffer is only sound when the consumer is done with
buffer N before the producer swaps back to it, and here the consumer is `subscriberBuffer` slots
deep across every subscriber, with `json.Encoder` on the far end of a handler that huma gives five
seconds to write. Handing the slice over and allocating a fresh one costs two allocations a second
and cannot corrupt a window mid-encode.

## `generation`, and the race it exists for

`flush` takes the generation its timer was armed for and returns when it no longer matches. Without
it: window A is open with timer T1; T1 fires and blocks on `s.mu`; the last subscriber leaves and
`discard` empties the window; a new client subscribes and a delivery arms T2; T1 finally gets the
lock, clears `s.timer` — which is **T2's** handle — and ships window B immediately. `close` then has
no handle to stop. It is the same fix, for the same shape of bug, as `idle_gate.deactivate`.

Comparing `s.timer == t` instead is not an option: `t` is assigned after the closure that would read
it is built.

`generation` is bumped on every arm, on the last unsubscribe and in `close`. A stale generation
therefore implies whoever bumped it already emptied the window, so an early return can never orphan
a non-empty `pending` with no timer armed.

## Fanning out under the lock is load-bearing

`flush` sends while holding `s.mu`, and `Unsubscribe` closes a subscriber's channel under the same
lock. Snapshotting the map and sending outside the lock — the instinctive "don't hold a lock while
sending" move — reintroduces a send on a closed channel. The sends are `select`/`default` and never
block, so the lock is held for microseconds twice a second.

`subscriberBuffer` is 8, not 64: at two windows a second that is four seconds of backlog, which
brackets huma's five-second `sse.WriteTimeout`. A bigger buffer only means a client that unsticks
has to replay half a minute of stale windows. A subscriber whose buffer is full loses the window and
is warned about **once**, with the total reported when it leaves.

## The hot path

Three regimes, and only the third costs anything:

1. **Nobody connected.** `binding.withOperation` sees `!Connected()` and takes the plain chain, so
   `Record` is never called. The decorator adds one interface hop to `Connected()`.
2. **Ring on, no SSE subscriber** — the idle tail after a dashboard closes, or a client that only
   hit `/api/operations`. `Record` is one delegation plus `watched.Load()`, one atomic load, no
   lock, no allocation. `TestStreamerRecordDoesNotAllocateWithoutSubscribers` pins the zero.
3. **Subscribed.** One uncontended mutex and a five-word struct copy per delivery.

`watched` is a **hint**, exactly like `Connected()`: a subscriber that appears mid-window misses the
deliveries recorded before the store is visible. Do not try to close that gap.

`Record` reads the five strings **before** delegating, so nothing here depends on what the ring's
buffer swap leaves behind — see `adapter/operations_collector/AGENTS.md`, "The buffer swap".

## The scrape belongs to `flush`, not to `Record`

A closing window asks the metrics collector for an out-of-schedule `Scrape`, so the dashboard's
numbers are fresh when the live event lands next to them. It happens in `flush` rather than per
delivery because `Scrape` takes the metrics collector's `mu` — the one `Dump` holds for as long as
it takes to merge a snapshot — and that has no business on the delivery path. The collector debounces
requests to one a second anyway, so a per-delivery call would throw away all but one in a couple of
thousand.

`Scraper` is declared here rather than imported so this package depends on the one method it uses.
It is nil when telemetry is disabled.

## Shutdown

`New` registers at `StageWorkers`, which runs before `StageServers`. Within a stage, hooks run in
**reverse registration order**, and `operationsCollectorAdapter.New()` is evaluated as an argument to
`New` — so the collector registers first and the streamer therefore tears down first. Publishing
stops before the ring is dropped. A refactor that hoists the inner `New` into an earlier statement
inverts that silently.

`StageServers` would be wrong: the web server's own hook registers later inside `Serve`, so reverse
order would run `server.Shutdown` first and leave it waiting on SSE handlers that have not been told
to leave. Closing the subscriber channels is what lets a parked handler return through its `!ok`
branch.

`close` is idempotent, and a `Record` after it still reaches the wrapped collector — the feed dies,
the log does not.

## Tests

Helpers live below the last test function — see the `go` skill. `newTestStreamer` uses a
millisecond window and `t.Cleanup(s.close)` to bypass the global shutdown hook, like
`newTestCollector` next door. `fakeCollector` counts what was delegated to it; its `records` counter
is atomic only because `TestStreamerSurvivesConcurrentRecordsAndSubscribers` writes it from four
goroutines.

`TestStreamerIgnoresAStaleTimer` drives `flush(generation-1)` directly, since the real race is not
reproducible on demand. `TestStreamerOpensANewWindowAfterAFlush` is what fails if anyone
reintroduces the double buffer.
