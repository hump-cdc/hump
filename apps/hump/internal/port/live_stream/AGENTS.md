# port/live_stream — how this package hangs together

The live operations feed of the dashboard: `Operation`, `Batch` and the `Streamer` a web client
subscribes to. No imports at all — not even `domain`; the fields are plain strings because they
go to JSON as they are. The implementation, and why it has no background goroutine, is in
`adapter/live_streamer/AGENTS.md`.

The package is **`livestream`**.

## The contract

- **One implementer, one consumer.** `adapter/live_streamer.Streamer` implements it;
  `hump_tower/stream_updates` is the only caller of `Subscribe` / `Unsubscribe`.
- **The port has no write side.** Deliveries enter through
  `port/operations_collector.Collector.Record`: the adapter decorates that collector, and
  `cmd/hump/main.go` hands the same value out under both interfaces
  (`operations, live = streamer, streamer`). Do not add a `Publish` here — the pipeline must keep
  writing to one collector.

## `Operation` is deliberately thin

Anything a client needs beyond where a delivery went comes from the operations log
(`port/operations_collector`). `Target` is the record's **rendered**
`Target`, not its `TargetKey` — so with a templated topic or URL it can differ from the target
the metrics topology is keyed by.
