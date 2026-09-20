# adapter/operations_collector — how this package hangs together

The dashboard's operations log. Where `metrics_collector` answers "what has the pipeline been
doing", this one answers "what happened to *that* row": one entry per delivery of one row to one
sink, holding the rendered message, the real target and every attempt it took. It is a ring of
fixed capacity, so the answer is always about the newest traffic and the memory is bounded.

## Layout

| File | Role |
|---|---|
| `operations_collector.go` | the `Collector`: options, lifecycle, `Connected`, `Record`, `Dump` |
| `ring.go` | the ring itself — `write` with the buffer swap, truncation, `page` |
| `logger.go` | the package logger scope |

## Nothing at all until a client connects

`Connect` allocates the ring; `Disconnect` on the last client arms the idle timer; `deactivate`
sets `ring = nil`. **Nothing survives that** — unlike `metrics_collector`, which deliberately
keeps `recorded` across an idle period. An idle collector here has to cost nothing, so there is
nothing to keep. `generation` plays the same role as in `metrics_collector`: an idle timer armed
before a reconnect cannot deactivate the collector after it.

There is **no background goroutine**. The collector is purely push-driven, so there is no scrape
loop, no ticker and nothing for goleak to find.

`active atomic.Bool` is the only atomic, and it exists for `Connected`, which the pipeline calls
on every single delivery and which therefore must not take `mu`. It is a **hint, not a
guarantee**: the last client can leave between the check and the `Record` that follows, so
`Record` returns silently when `ring == nil`. Do not turn that into an error.

## One record per delivery, retries inside it

A retry is an element of `Record.Attempts`, never a record of its own. `Record.Status` is the
terminal outcome (`success` or `failure`); `Attempt.Status` uses the whole
`domain.DeliveryStatus` vocabulary, so a delivery that failed twice and then went through is one
record with `Status: success`, `Retries: 2` and attempts `retry, retry, success`.

An attempt's status is only knowable once the retry loop is done — a failed attempt is a `retry`
unless it turns out to be the last — so `process_operation.operation.end` labels them all at the
end rather than as they happen. That is also why `pkg/resilience` is untouched: its hooks carry
neither the error nor the attempt index, and `binding` is shared across deliveries, so nothing
useful can be correlated through them.

## The buffer swap is the whole performance story

`Record` does not copy the payload. It **swaps**: the slot takes the caller's `Payload`,
`Metadata` and `Attempts`, and the caller gets the slot's previous ones back. The writer refills
those on the next delivery, so `Record` itself allocates nothing at all in steady state.
`TestRingWriteDoesNotAllocateInSteadyState` pins that with `testing.AllocsPerRun`, and
`BenchmarkRecordActive`/`BenchmarkRingWrite` report it. All three need a priming lap: until
every slot has been written once, the buffers handed back are the zero value's nil ones.

Scope that claim to `Record`. The **delivery** path, while connected, is not allocation free:
every `Metadata` entry concatenates its `header.` prefix, and HTTP formats its status code.
Those are the price of the one-map-per-record shape and cannot be avoided under it. The measured
cost is one allocation per header over the disconnected baseline —
`BenchmarkBindingProcessLogged` against `BenchmarkBindingProcessWithoutLog` — and none of it is
paid when nobody is watching.

Two consequences a caller must respect, both stated on `port.Collector.Record`:

- the caller **must not retain** anything it passed in, and must re-point at what came back —
  `process_operation.operation.rebind` is exactly that, and without it the second delivery
  appends into the record the ring already stored;
- `Dump` therefore **deep-copies** every buffer. The slots are overwritten the moment recording
  resumes, so handing out the ring's own slices would be a data race with a reader.

`payloadLimit` (64 KB by default) bounds the ring at `capacity × limit` ≈ 64 MB. An oversized
payload is copied down to the limit — the one allocation on that path, pinned by
`TestRingWriteAllocatesOnceForAnOversizedPayload` — and its buffer is dropped rather than kept,
so a single wide row cannot pin memory for the life of the process. `PayloadSize` keeps the
untruncated length. A payload of exactly the limit is not copied at all.

`next` wraps with a comparison rather than `%`: the capacity is a user option and is not
required to be a power of two.

## Paging counts back from the newest

`Dump(start, end)` treats offset 0 as the newest record. Both offsets are clamped to `Total`, so
an out-of-range window — including every window on an idle collector — is an empty page with
honest `Total` and `Written`, not an error. `Written` counts everything recorded since the last
activation, including what has already been overwritten, which is how a reader notices it fell
behind.

`Seq`, unlike `Written`, **carries across an idle period**: `stop` saves it on the collector and
the next `activate` resumes from it. A reader holding a `Seq` as a cursor would otherwise see
the numbering restart after a reconnect and re-read records it had already seen.

## Who feeds it

`process_operation.binding`, through the `OperationLog` interface declared there. It builds
**two** delivery chains, not one with a flag: the logged chain writes into the binding's scratch
record from every attempt, so the choice has to be made once, outside the retry loop.
`withOperation` picks between them, and the `inflight` CAS covers the case where the worker
pool's one-delivery-per-binding invariant is ever broken — such a delivery is still delivered,
just not logged.

The sink adapters fill a `sink.Capture` **after publishing, on both the success and the error
path**: a failed delivery is the one whose request a reader most wants. Headers and everything
else sink-specific go into the single `Metadata` map under a `header.` prefix, so a new sink type
never needs the port widened.

## Tests

`operations_collector_test.go` covers the collector through its port; `ring_test.go` covers the
ring directly, which is where the buffer swap is pinned by **identity** — `samePayload` and
`sameAttempts` compare backing arrays, so a test fails if the ring starts copying instead of
swapping. `newTestCollector` bypasses the global shutdown hook with `t.Cleanup(c.close)`;
`write` records one delivery; `waitFor` polls a condition for a second. Helpers live below the
last test function — see the `go` skill.

The allocation claims live in tests, not only in benchmarks: `AllocsPerRun` asserts 0 for a
steady-state `write`, 1 for a truncating one and 0 for an empty `page`, and
`TestRingWrite1MRecordsStaysBounded` writes a million records through a 1000-slot ring and
checks the heap did not grow with them. Those counts hold under
`-race` and `-cover`, so they need no build-tag escape hatch. Benchmarks are split the same way
as the tests — `bench_test.go` for the collector, `ring_bench_test.go` for the ring — and
`BenchmarkRingPage` is report-only: a page allocates per record it clones, and that count is not
a contract.

`BenchmarkBindingProcessWithoutLog` in `process_operation` is the control for
`BenchmarkBindingProcessIdle`: both must report the same allocation count, which is what proves
a disconnected log costs nothing. Neither is allocation free, and that is not this package's
doing — the per-attempt `context.WithTimeout` was always there.
