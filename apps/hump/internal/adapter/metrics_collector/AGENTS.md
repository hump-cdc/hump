# adapter/metrics_collector — how this package hangs together

The dashboard's metrics source. It scrapes the process registry on demand and answers one
question per request: what has the pipeline been doing, both over the last few minutes and
since it started. It is the only consumer of the registry inside the process — everything else
goes out over `/metrics`.

## Layout

| File | Role |
|---|---|
| `metrics_collector.go` | the `Collector`: lifecycle, the scrape loop, `Record`, `Dump` |
| `scrape.go` | gathered families → deltas plus the absolute readings (`latest`) |
| `bucket.go` | `aggregate` / `bucket` / `series` / `histogram` — the windowed side |
| `snapshot.go` | topology + readings + window → `port.Snapshot` |

## Idle by default

The collector does nothing until a client calls `Connect` (`stream_updates`, once per `/api/live`
connection). The refcount, the idle timer and the generation guarding it live in
`pkg/idle_gate` (see its `AGENTS.md`); this package supplies the two callbacks.
`stop` **drops every aggregate**: `scraper`, `series` and `lastTick` all go back to zero. A `Dump`
after that is not an error — `snapshot` reports the configured topology with zero counts, which
is the same shape an active collector with no traffic returns. Do not remove that guard; the
tests dump an idle collector on purpose.

The gate runs `activate` and `stop` under its own lock, so they must never call back into the
gate. `activate` spawns the scrape loop while both locks are held;
the new goroutine's first `tick` locks `c.mu` to take `waiting`, so it blocks until `activate`
returns, then gathers outside any lock.

`close` cannot be folded into `stop`: the gate calls `stop` only when the collector is active,
while `recorded` has to be cleared even on a collector that never connected — which is what
`TestRecordIgnoredAfterClose` pins.

`Scrape` requests an out-of-schedule scrape, debounced in `run`, and returns a channel the caller
can wait on — `live_streamer` holds a window back until it closes. The contract:

- The channel is `waiting`, shared by every `Scrape` until the next tick **starts**; that tick
  closes it when it is done, whether or not the gather succeeded.
- `tick` takes `waiting` *before* gathering. A `Scrape` that arrives mid-gather may follow a
  delivery the registry read has already missed, so it gets a new channel and waits for the next
  tick.
- An idle collector answers `closedCh`, already closed, and `stop` closes a pending `waiting`, so
  nobody waits on a loop that is gone.
- `Scrape` never blocks: `requests` has one slot and a full one is skipped.

## Two clocks in one response

This is the thing to get right before touching `snapshot.go`.

- **`Points` are windowed deltas.** Counters and histograms contribute how far they advanced
  since the previous scrape, folded into `Truncate`d buckets, `pointsMax` of them per width. A
  series seen for the first time only records a baseline, so the first scrape after activation
  contributes nothing but gauges.
- **`Tables`, `Sinks` and `Connections` are lifetime counts**, so they do not depend on the
  requested duration. `Tables` and `Sinks` report the absolute counter readings of the most
  recent scrape (`scraper.latest`); `latest` is rebuilt wholesale on every scrape, which is how
  a series the registry stopped exposing disappears. `Connections` is not scraped: `edges()`
  reads `c.recorded`, which `Record` fills (see below).
- **`p95` is windowed anyway**, because a lifetime latency quantile says nothing useful. It
  comes from `series.total()` — every retained bucket of the requested width merged into one
  aggregate — and is reported in **milliseconds**, while the histograms observe seconds.

## What it reads, and what it cannot

Five families, named in `scrape.go`:

| Family | Used for |
|---|---|
| `hump_sink_messages_total{db,status,type,host,target}` | `Points.Delivered`/`Failed`, `Sinks`, `Targets` |
| `hump_sink_duration_seconds{db,type,host,target}` | the `p95` of points, `Sinks` and `Targets` |
| `hump_processed_rows_duration_seconds{db,operation,schema,table}` | `Tables`: counts via `SampleCount`, `p95` via the buckets |
| `hump_replication_lag_bytes`, `go_memstats_alloc_bytes` | the gauge readings on a point |

`status` has three values, not two: `success`, `retry` and `failure` — the vocabulary is
`domain.DeliveryStatus`, shared with `process_operation`, which is what writes the labels. A
retry is an attempt rather than an outcome, so it is left out of every operation count and
every success rate.

Four caveats that are properties of the source metrics, not of this package:

- **Row counts miss failed rows.** `postgres.observeRow` only fires when the handler returned
  `nil`, so a row whose delivery ultimately failed never reaches `Tables`. Partitions also roll
  up into the configured parent, so `Tables` keys are configured tables, not relations.
- **Table `p95` is end to end.** For WAL rows `observeRow` times decoding plus
  `UseCase.Process`, which waits for every sink of the row, retries included — not a single
  sink write. For snapshot rows it times only the `onRow` callback, after the COPY parser has
  already decoded the row.
- **Sink `p95` is per attempt.** `OnDuration` is called inside the retry loop
  (`pkg/resilience/retry.go`), once per attempt including the ones that failed — the metric's
  own help text claims otherwise.
- **The sink metrics carry no table.** Two tables routed to the same `(type, host, target)`
  collapse into one series, which is why `Connections` cannot be scraped at all.

## Topology comes from outside, counts for it come from the pipeline

`WithTopology` takes the configured `table → sink → target` edges, resolved at wiring in
`cmd/hump/main.go`. The adapter must not import `hump/config`; it takes plain `port.Connection`
values. The resolver there renders host and target with
`Metadata(sink.Context{Schema, Table})` — **the same call `newBinding` makes** — because the
strings have to match the metric labels byte for byte. Change one side and the snapshot goes
silently empty.

The topology is also the key set: a configured table or target with no traffic is still
reported, with zeros.

`Connections` counts are pushed in through `Record`, called from `process_operation.binding`
next to the Prometheus counters. They are kept by status, so a connection reports its success
and failure counts, retries left out. Two consequences:

- `recorded` has **its own mutex**. `Record` runs per delivery on every worker goroutine, while
  `Dump` holds `c.mu` long enough to merge twenty buckets per sink key. Lock order is `g.mu`
  (the idle gate) then `c.mu` then `recordedMu`, never the reverse. `close` takes `recordedMu`
  only after the gate has released `g.mu`.
- `recorded` **survives deactivation**, unlike everything else the collector holds. It sits next
  to lifetime counters in the response and would be confusing if it reset on an idle timeout.
  Only `close` clears it.

## Tests

`countingGatherer` counts registry reads so a test can assert the loop is or is not running.

A latency test has to **observe once before the collector starts**: the first scrape is the
baseline, so observations made before it are invisible. Prime the series, wait for the first
point, then observe what the test measures.
