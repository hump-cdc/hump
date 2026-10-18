# application/hump_tower/get_snapshot_estimates — how this package hangs together

`POST /api/snapshot/estimate` (`snapshot.estimate`, registered in
`internal/controller/hump_tower.go`), an SSE stream despite the verb: the body lists tables with
their filters, and row counts come back as they arrive — for each table the planner's estimate first
(`rough: true`), then the exact count, which is dropped silently if it fails or times out. It feeds
the snapshot dialog while the user is still editing filters, so it is independent of any run and
safe to call at any time.

## Layout

| File | Role |
|---|---|
| `usecase.go` | `Estimate`: one goroutine per table, a slot semaphore, one shared channel |
| `controller.go` | the SSE handler, `Events()`, request and event conversion |
| `dto.go` | `EstimatesRequest` and the three event payloads — the wire contract |

## Concurrency

`Estimate` starts a goroutine for **every** requested table at once and returns an unbuffered
channel that closes when all of them are done. What is bounded is the counting: a goroutine takes a
slot from `slots` before it calls `GetRowCount`, and `defaultMaxConcurrentEstimates` is 6 because
each count holds a database session of its own for as long as it runs
(`adapter/postgres/postgres_row_count.go`). `WithMaxConcurrentEstimates` overrides it and ignores a
non-positive value.

Events of different tables interleave in arrival order. Events of **one** table keep the order
`GetRowCount` produced them in, which is what makes "rough, then exact" reliable for the client.

## Nobody may block on a reader that left

This is the invariant the whole shape serves; the `…Stops…` and `…Cancelled` tests in both files
pin it.

- The handler wraps its context in `WithCancel` and cancels on **return** — including the early
  return after a failed `send`. Without that, a client that drops mid-stream leaves goroutines
  parked on the unbuffered channel forever, each holding a session.
- Every send goes through `emit`, a `select` against `ctx.Done()`; so does waiting for a slot.
- After a slot is acquired, `ctx.Err()` is checked **again**: when a slot frees up after the
  context is already done, `select` picks between the two ready cases at random, and half the time
  it would start a query nobody is waiting for.

The handler's two exits differ: a failed `send` returns at once without `done`; a cancelled request
drains the channel, which closes, and then attempts `done` anyway.

## Failures are events, not a failed stream

A table that cannot be counted does not end the stream. Both an error from `GetRowCount` itself (an
invalid filter, a missing table) and a `RowCount.Error` arriving mid-stream (a failed query) become
an `error` event for that table, and the other tables carry on.

`schema` and `table` are echoed **as requested**, not as resolved: an empty schema resolves through
`search_path` on the server and stays empty in the event, since the client matches events to its
rows by what it sent.

A nil database closes the channel at once, so the stream is a lone `done`. That path exists for
`cmd/openapi`, which builds the use case with nil to register the route, and for tests.

## Changing an event touches two places

`Events()` maps names to payload types for huma, but `cmd/openapi/main.go` separately lists the
three schema names for this route in `flattenSSESchema`, which rewrites the documented response
into a `oneOf` of payloads for the generated TypeScript client. Two consequences:

- A new or renamed event must be added there too, or the client never sees its type.
  `TestEventsNamesEveryPayload` pins the `Events()` map, not that list.
- The flattened spec **drops event names**, so payloads have to stay distinguishable by shape:
  `count` + `rough`, versus `error`, versus the empty `done`.

## Tests

`usecase_test.go` drives `Estimate` against `fakeDatabase`, which scripts counts and errors per
table name, can `delay` or `block` a count, and tracks the peak number of concurrent calls for
`TestEstimateCapsConcurrentCounts`. `controller_test.go` drives the handler through `fakeSender`
(`failAfter` simulates a client that left) and `serve`, which fails a test that runs past five
seconds. Assertions on events use `sameElements` — arrival order across tables is not a contract.
Both files share one package, so helpers such as `scriptTables`, `identifier`, `waitFor` and
`collect` are defined once, in `usecase_test.go`. Helpers live below the last test function — see
the `go` skill.
