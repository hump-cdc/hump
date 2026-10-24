# port/operations_collector — how this package hangs together

The dashboard's operations log as its consumers see it: `Record` and `Attempt` going in, `Query`
and `Page` coming out, and the `Collector` between them. One file, `operations_collector.go`.
The ring, the buffer swap and the filter are in `adapter/operations_collector/AGENTS.md`.

The package is **`operationscollector`**.

## Two implementers, one of them a decorator

`adapter/operations_collector.Collector` is the log. `adapter/live_streamer.Streamer` implements
the same interface by delegating every method to the collector it wraps, and that wrapper is
what `cmd/hump/main.go` hands out. A method added to `Collector` has to be added to the
decorator as well, or the wiring stops compiling.

## Who calls what

- **`process_operation`** needs `Connected` and `Record` only, and declares its own
  `OperationLog` interface for them rather than taking the `Collector`.
- **`get_operations`** calls `Search`; **`stream_updates`** calls `Connect` / `Disconnect`,
  which are a refcount and must be paired. `Dump` has no caller in the application layer.

## `Record` transfers ownership

The ownership rule is on `Collector.Record`, and it is the part of the contract a caller breaks
most easily: it must not keep a reference to what it passed in, and must continue with what came
back. That is why the parameter is a pointer.

## Vocabulary that is easy to get wrong

The field comments define each term. One of them is also a wire format:
`Record.Status` is `success` or `failure` only, and `get_operations/dto.go` repeats that as
`enum:"success,failure"` for the status filter. Change both together.

`SinkType`, `Host` and the `SinkTypes`, `Hosts`, `Targets` and `Tables` filters are plain strings:
this package imports `domain` only, not `port/sink`.
