# adapter/operations_collector — how this package hangs together

The dashboard's operations log. Where `metrics_collector` answers "what has the pipeline been
doing", this one answers "what happened to *that* row": one entry per delivery of one row to one
sink, holding the rendered message, the real target and every attempt it took. It is a ring of
fixed capacity, so the answer is always about the newest traffic and the memory is bounded.

## Layout

| File | Role |
|---|---|
| `operations_collector.go` | the `Collector`: options, lifecycle and the gate callbacks, the port methods |
| `ring.go` | the ring: `write` with the buffer swap and truncation, `search` with its `matcher`, `clone` |

## Nothing at all until a client connects

The refcount, the idle timer and the generation that guards it all live in `pkg/idle_gate` (see
its `AGENTS.md` for the guarantees). This package supplies only the two callbacks: `activate`
allocates the ring, `stop` sets `ring = nil`. **Nothing survives that** except `lastSeq` — unlike
`metrics_collector`, which deliberately keeps `recorded` across an idle period. An idle collector
here has to cost nothing, so there is nothing to keep.

Lock order is `g.mu` then `c.mu`, never the reverse; nothing that holds `c.mu` may reach the gate.

The gate clears its active flag **after** `stop` returns, which is what makes
`Connected() == false` imply `ring == nil`. Invert that order and
`TestCollectorDropsRingAfterIdleTimeout` starts failing intermittently rather than outright.

There is **no background goroutine** of this package's own. The collector is purely push-driven,
so there is no scrape loop and no ticker; the only goroutine goleak can find is the gate's idle
timer, and it is gone once the collector deactivates.

`Connected` is `gate.Active`, one atomic load, because the pipeline calls it on every delivery.
It is a hint: the last client can leave between the check and the `Record` that follows, so
`Record` returns silently when `ring == nil`. Do not turn that into an error.

## One record per delivery, retries inside it

A delivery that failed twice and then went through is one record with `Status: success`,
`Retries: 2` and attempts `retry, retry, success`.

`Record.Error` is the terminal error of the retry loop. It is not always the last attempt's text:
a failure followed by a backoff that finds
the stop already closed is wrapped in `ErrRetryAborted`, which is the more useful string of the
two. A stop that closes during the wait instead cuts it short and runs one more attempt at once;
that attempt's error is wrapped only if attempts remain, and comes back bare on the last one
(`pkg/resilience/retry.go`).

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
`BenchmarkRingWrite` reports it. Both need a priming lap: until
every slot has been written once, the buffers handed back are the zero value's nil ones.
`ring_test.go` also pins the swap by **identity** — `samePayload` and `sameAttempts` compare
backing arrays, so a test fails if the ring starts copying instead of swapping.

Scope that claim to `Record`. The **delivery** path, while connected, is not allocation free:
the sinks allocate while filling `Metadata` (string conversions, the `header.` prefix). That is
the price of the one-map-per-record shape, and none of it is paid when nobody is watching.
No benchmark measures that per-header cost: the `fakeSink` behind the `process_operation`
binding benchmarks writes no `header.*` entry.

Two consequences a caller must respect, stated on `port.Collector.Record` and `Dump`
respectively:

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

`TestRingWrite1MRecordsStaysBounded` writes a million records through a 1000-slot ring and checks
the heap did not grow with them. The allocation counts hold under `-race` and `-cover`, so they
need no build-tag escape hatch. `BenchmarkRingPage` is report-only: a page allocates per record
it clones, and that count is not a contract.

## Paging and the filter

`Search(Query)` is the filtered `Dump`; `Dump` is `search` with an empty query. The
filter has to live here rather than in the web controller, because the alternative is dumping
`capacity × payloadLimit` — 64 MB by default, every buffer deep-copied — to throw most of it
away. `search` instead counts every match into `Total` and clones only the ones inside
`[Start, End)` **of the matches**, so offset 1 is the second match, not the second record.

The order is **strictly newest first**, filter or no filter: `index` walks offsets back from
`next - 1`. Nothing downstream re-sorts, and `TestOperationsRouteAnswersNewestFirst` pins it at
the HTTP edge as well as `TestRingSearchWithoutFiltersReadsEverything` in the ring. The API's
`from`/`to` map straight onto `Start`/`End`, so `from: 0, to: 50` is the 50 newest matching
records.

The whole ring is walked even once the window is full, because `Total` has to be honest — it is
what tells a client how far it can page. `Written` counts everything recorded since the last
activation, including what has already been overwritten, which is how a reader notices it fell
behind.

`Seq`, unlike `Written`, **carries across an idle period**: `stop` saves it on the collector and
the next `activate` resumes from it. A reader holding a `Seq` as a cursor would otherwise see
the numbering restart after a reconnect and re-read records it had already seen.

`Targets` matches `Record.TargetKey`, not the rendered `Target`, which is what lets the flows
inspector list the deliveries of one node. Matching a record allocates nothing:
`newMatcher` lowercases `Query.Text` once, before the scan, and
`matchesTable` compares each `"schema.name"` in place against the record's schema, a dot and
its name — the `schema + "." + name` a naive comparison
would build is exactly the allocation this avoids. `containsFold` is generic over
`~string | ~[]byte` for the same reason: a payload needs no conversion, and
`strings.Contains(strings.ToLower(…))` would allocate per record.
`TestRingSearchDoesNotAllocateWhenNothingMatches` and
`TestRingSearchDoesNotAllocateWithListFilters` pin that at 0.

The cost is the lock: `Search` holds `mu` for the full walk, and with a `Text` set that is up to
`capacity × payloadLimit` bytes scanned while `Record` waits. Acceptable for a dashboard query,
but it is the reason not to expose this to anything that polls hard.

## Who feeds it

`process_operation.binding`, through the `OperationLog` interface declared there. It builds
**two** delivery chains, not one with a flag: the logged chain writes into the binding's scratch
record from every attempt, so the choice has to be made once, outside the retry loop.
`withOperation` picks between them, and the `inflight` CAS covers the case where the worker
pool's one-delivery-per-binding invariant is ever broken — such a delivery is still delivered,
just not logged.

The sink adapters fill `Metadata` through `sink.Capture`; the keys are in `adapter/sink/AGENTS.md`
and the `capture.Set` calls.

`BenchmarkBindingProcessWithoutLog` in `process_operation` is the control for
`BenchmarkBindingProcessIdle`: both must report the same allocation count, which is what proves
a disconnected log costs nothing. Neither is allocation free, and that is not this package's
doing — the per-attempt `context.WithTimeout` was always there.

## Tests

The gate's own mechanics — refcount, idle timer, stale generations — are covered in
`pkg/idle_gate`, not here; what stays is what the callbacks do to the ring.
`newTestCollector` bypasses the global shutdown hook with `t.Cleanup(c.close)`.
