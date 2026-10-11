# shared/prometheus — how this module hangs together

A client-side reader for the Prometheus text exposition: parse a scrape, pick families and
series out of it, and turn cumulative counters and histogram buckets into deltas and quantiles.
Pure TypeScript — no React, no fetch, no storage, and no imports outside its own folder (the
tests add `bun:test`). The caller supplies the scrape text and keeps the previous readings.

**Nothing imports it today.** No file under `web/webui/src` outside this folder references
`@/shared/prometheus`. The dashboard's numbers come from `POST /api/metrics` through
`entities/metrics` (`useMetrics`, built on `@/shared/api`), where the same work — scraping,
deltas, `p95` — is done in the process by `internal/adapter/metrics_collector` and rendered by
`internal/application/webui/get_metrics`; read their `AGENTS.md` first. Before building on this
module, decide whether the number belongs in that response instead.

## Layout

| Path | Role |
|---|---|
| `index.ts` | the public contract; `biome.json` forbids importing `@/shared/prometheus/**` past it |
| `types.ts` | the data types every stage exchanges |
| `metrics.ts` | `FAMILY` — the tracked family names — and what is derived from it |
| `parse/parser.ts` | `createParser(tracked)` → `(text) => PromMetrics` |
| `parse/series-key.ts` | `keyOf(sample)` — the canonical series identity |
| `query/samples.ts` | `samplesOf`, `seriesOf`, `sumOf`, `gauge` — selection over `PromMetrics` |
| `calc/counters.ts` | `readingsOf`, `counterWindow` — baselines and deltas |
| `calc/histogram.ts` | `parseBound`, `mergeBuckets`, `quantile` |

## The pipeline

```ts
const parse = createParser(TRACKED_METRICS)

const metrics = parse(text)                                   // PromMetrics
const window = counterWindow(previous, restarted)             // previous: CounterSnapshot
const delivered = window.delta(
    seriesOf(metrics, FAMILY.sinkMessages, ({ status }) => status === STATUS_SUCCESS),
)
const next = readingsOf(metrics, COUNTER_FAMILIES)            // the next poll's `previous`
```

| Stage | Takes | Returns |
|---|---|---|
| parse | exposition text | `PromMetrics`: family name → `PromSample[]` in scrape order |
| query | `PromMetrics`, a `MetricFamily`, an optional `LabelMatch` | `PromSample[]`, `PromSeries[]` (`{ key, value }`), a sum, or one gauge value |
| calc | `PromSeries[]` + a `CounterSnapshot`; or `HistogramBucket[]` | deltas; a quantile |

Nothing in the module holds state between scrapes. `previous` and the `restarted` verdict are
the caller's: `FAMILY.processStart` is tracked so a restart can be detected, but no code here
compares it.

There is also no function that turns `_bucket` samples into `HistogramBucket[]`. The caller
builds them — `le` through `parseBound(labels.le)`, `count` from the sample or its delta — and
must hand `quantile` a list that is cumulative and ascending by `le`.

## What the parser reads

The parser is an allowlist filter first. Names are indexed by length and matched as whole
names, so an untracked line is rejected before anything is sliced; a matched sample's `name` is
the tracked constant itself, not a copy. It keeps only what `tracked` lists — `createParser`
takes the list so the parser knows nothing about hump.

Handled, each pinned by `parser.test.ts`:

- `name value` and `name{labels} value`, an empty `{}`, spaces around `=` and `,`;
- label escapes `\\`, `\"`, `\n`; a `}`, `,` or `=` inside a quoted value does not end the list;
- an optional trailing timestamp, which is dropped;
- CRLF, blank lines, a final line with no newline.

Dropped silently, never thrown: `#` lines (so `HELP` and `TYPE` are not read and no family has
a type), a line with no value, a label list that is truncated or malformed, and a value that is
not finite — `NaN`, `+Inf`, `-Inf`.

Because `TYPE` is ignored, a histogram is not one family here. Its `_bucket`, `_count` and
`_sum` lines are separate names, and only the ones spelled out in `FAMILY` survive:
`FAMILY.rows` is `…_seconds_count` and `FAMILY.sinkDuration` is `…_seconds_bucket`.

## Selection and identity

A `LabelMatch` is a predicate over the label record; there is no selector syntax. With no
predicate the whole family matches. An absent family is `[]` for `samplesOf` / `seriesOf`, `0`
for `sumOf` and `undefined` for `gauge`. `gauge` reads the **first** sample and ignores labels,
so it only fits an unlabeled single-sample family.

`keyOf` renders `name{k="v",…}` with label names sorted, so the same series matches across
scrapes whatever order its labels arrive in (`counters.test.ts`, "regardless of label order").
It writes the result back onto the sample (`PromSample.key`) the first time it is asked.
`CounterSnapshot` is keyed by this string: change the rendering and every baseline a caller
kept stops matching. Only `"` and `\` are re-escaped.

## Deltas

`counterWindow(previous, restarted)` decides the baseline of every series once per scrape:

| Case | `delta` adds | `perSeries` |
|---|---|---|
| has a previous reading, value ≥ it | `value - previous` | same |
| value **below** its previous reading | the whole `value` | same |
| no previous reading | `0` | **key absent** |
| `restarted` is true | the whole `value`, for every series | same, unseen series included |

- `perSeries` leaves an unmeasured series out so a caller can tell "did not advance" (`0`) from
  "not measured" (absent). `delta` cannot make that distinction.
- A drop is treated as a reset even without `restarted`. The flag exists for the reset the
  readings hide: a counter that climbed back past its old value within one poll.
- `readingsOf` takes the families explicitly. Pass `COUNTER_FAMILIES`; a gauge has no baseline.

## Quantiles

`quantile(buckets, q)` is linear interpolation inside the bucket the rank `q × total` falls in,
with `total` read off the **last** bucket and the first bucket's lower bound taken as `0`.

- `undefined` when the total is `0` or less, or when the rank lands in the `+Inf` bucket —
  "no estimate", not zero.
- A bucket holding no observations of its own returns its lower bound.

`mergeBuckets(into, from)` sums counts bound by bound and returns a new, ascending list without
touching its arguments. A bound present in only one list keeps that list's count alone, so the
result is cumulative only when both lists share their bounds.

## `metrics.ts` and the exporter

`FAMILY` is the single list: `MetricFamily` and `TRACKED_METRICS` are derived from it, and every
query function takes a `MetricFamily`, so a mistyped name fails to compile. `COUNTER_FAMILIES`
is maintained by hand next to it.

The names are copies of what the process exposes at `/metrics`. `sink_messages_total` and
`sink_duration_seconds` are declared in `internal/application/processing/process_operation`,
`processed_rows_duration_seconds` in `internal/adapter/postgres/metrics.go`; the `hump_` prefix
and the Go and process collectors come from `pkg/telemetry/metric/metric.go`.
`internal/adapter/metrics_collector/scrape.go` names the same three families for the in-process
scrape. When one of them is renamed, change `FAMILY`, add a new cumulative family to
`COUNTER_FAMILIES`, and check `scrape.go` — a stale name here is not an error, just a family
that never appears.

That has already happened: `hump_lag_commit_seconds`, `hump_lag_sender_seconds` and
`hump_async_queue_total` are not declared anywhere in the Go code, and `series-key.ts` points at
`internal/telemetry/metrics/metrics.go`, which does not exist.

`STATUS_SUCCESS` / `STATUS_FAILURE` are two of the three `status` values; `retry` is left out on
purpose, as it is in `metrics_collector`.

## Tests

`bun test` from `web/webui` (the `test` script); `bun test src/shared/prometheus` runs this
module alone. Tests sit next to the code: `parser.test.ts`, `counters.test.ts`,
`histogram.test.ts`. `query/` and `series-key.ts` have none of their own — `counters.test.ts`
exercises them through `seriesOf` and `readingsOf`.
