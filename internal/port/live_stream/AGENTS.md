# port/live_stream — how this package hangs together

The live operations feed of the dashboard: `Operation`, `Batch` and the `Streamer` a web client
subscribes to. One file, `live_stream.go`, and no imports at all — not even `domain`; the fields
are plain strings because they go to JSON as they are.

The directory is `live_stream`, the package is **`livestream`**, so every import needs an alias.

## Declarations only

This package holds no behaviour and must stay that way. A `FanOutStreamer` with a permanent
goroutine used to live here and was removed; the implementation is `adapter/live_streamer`, and
`adapter/live_streamer/AGENTS.md` explains the window, the timer and why there is no background
goroutine.

## The contract

- **One implementer, one consumer.** `adapter/live_streamer.Streamer` implements it;
  `webui/stream_updates` is the only caller of `Subscribe` / `Unsubscribe`.
- **The port has no write side.** Deliveries enter through
  `port/operations_collector.Collector.Record`: the adapter decorates that collector, and
  `cmd/hump/main.go` hands the same value out under both interfaces
  (`operations, live = streamer, streamer`). Do not add a `Publish` here — the pipeline must keep
  writing to one collector.
- **Lossy by design.** A subscriber that cannot keep up loses batches; the pipeline is never
  slowed down for a browser.
- **A closed streamer answers a closed channel**, so a caller that subscribes during shutdown
  falls out of its `range` at once instead of hanging.
- **Nil is a valid `Streamer`.** It is wired only when the web UI is enabled, and
  `stream_updates` nil-checks it.

## `Operation` is deliberately thin

Where a delivery went and nothing else: no payload, no timing, no status. Anything a client
needs beyond that comes from the operations log (`port/operations_collector`). The adapter
copies the fields from the `Record` it was handed, and `Target` is the record's **rendered**
`Target`, not its `TargetKey` — so with a templated topic or URL it can differ from the target
the metrics topology is keyed by.
