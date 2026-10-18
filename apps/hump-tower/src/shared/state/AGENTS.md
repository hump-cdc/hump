# shared/state — how this module hangs together

The TanStack Query key registry and the two invalidation routines the live stream runs. No store,
no hooks, no React: plain functions over a `QueryClient`. FSD layer: `shared`. Its only import
from the app is type-only, `MetricsRequestBodyModel` and `OperationsRequestBodyModel` from
`@/shared/api`.

Consumers: `entities/metrics`, `entities/operations`, `entities/flows`, `features/snapshot` take
`keys`; `shared/live` takes the two `invalidateFor*` functions.

## Layout

| File | Role |
|---|---|
| `keys.ts` | `keys` — every hand-written query and mutation key — and `OperationsFilterKey` |
| `invalidate.ts` | which queries a live batch and a reconnect make stale, and the call that does it |
| `invalidate.test.ts` | pins the filters, the `cancelRefetch` option and prefix matching on real keys |
| `index.ts` | exports all of the above, the two `LIVE_*_QUERIES` filter lists included |

## The key scheme

| Key | Value | Used by |
|---|---|---|
| `keys.queries.metrics(duration)` | `['metrics', duration]` | `entities/metrics` |
| `keys.queries.operations(filter, block)` | `['operations', filter, block]` | `entities/operations` |
| `keys.queries.recentOperations(filter)` | `['operations', 'recent', filter]` | `entities/flows` |
| `keys.queries.snapshotTables()` | `['snapshot', 'tables']` | `features/snapshot` |
| `keys.queries.snapshot()` | `['snapshot', 'run']` | `features/snapshot` |
| `keys.queries.snapshotEstimate(schema, table, where)` | `['snapshot', 'estimate', schema, table, where]` | `features/snapshot` |
| `keys.prefixes.metrics()` / `.operations()` | `['metrics']` / `['operations']` | `invalidate.ts`, `entities/operations` |
| `keys.mutations.snapshotStart()` / `.snapshotStop()` | `['snapshot', 'start']` / `['snapshot', 'stop']` | `features/snapshot` |

- **The key carries the request.** `entities/operations` reads the filter and the block index back
  out of the key in its `queryFn` and builds the body from them. `OperationsFilterKey` is the
  request body without `from`/`to` because the block index stands for that window.
- **Prefixes exist for invalidation.** A prefix matches every variant under it: `['operations']`
  covers both the paged list and the `recent` list. The roots are module constants (`METRICS`,
  `OPERATIONS`) so that `keys` does not reference itself while it is being initialised.
- The snapshot keys share `'snapshot'` as first element but there is no `prefixes.snapshot`.
- **Not everything is here.** The config query uses the generated `getApiConfigOptions()` from
  `@/shared/api`, whose key is an object (`[{ _id: 'getApiConfig', baseUrl }]`), so it sits
  outside this scheme and outside every prefix.

## Invalidation

`shared/live/live-updates.tsx` (`LiveUpdates`, mounted once in `src/app/providers.tsx`) is the
only caller:

| Trigger | Function | Filters |
|---|---|---|
| a non-empty `operations` batch from `/api/live` | `invalidateForLiveEvent` | `['metrics']` |
| the stream starting a second or later attempt | `invalidateForLiveReconnect` | `['metrics']` with `refetchType: 'all'`, then `['operations']` |

- A live event says only that deliveries happened (see
  `apps/hump/internal/application/hump_tower/stream_updates/AGENTS.md`); the numbers come from
  refetching.
- On an event only mounted metrics queries refetch (the react-query default). After a reconnect
  events were missed, so metrics refetch **even while unmounted**; operations are only marked
  stale.
- Operations are deliberately absent from the per-event list: `entities/operations` and
  `entities/flows` subscribe to the stream themselves and invalidate their own keys through a
  debounce. Do not add `['operations']` to `LIVE_EVENT_QUERIES`.
- **`cancelRefetch: false` is load-bearing.** The default `true` cancels an in-flight refetch and
  starts a new one; at about two events a second a request slower than that would restart
  forever. A test asserts the option on every call. The two entities pass the same option in
  their own invalidations.
- `invalidate` fires the promises and does not await them.

## Adding a key

1. Add a factory under `keys.queries` or `keys.mutations`, returning an `as const` tuple that
   starts with the resource name and contains every input the request depends on.
2. If the query must refresh on live traffic, give it a root constant and a `keys.prefixes` entry,
   build the key from that root, and decide which list in `invalidate.ts` it belongs to: per event
   (cheap, mounted only) or reconnect only. Extend `invalidate.test.ts` — it asserts the exact
   number and order of calls.
3. Never write a key literal in a slice; `invalidate.ts` can only reach keys built from these
   roots.

## Tests

`bun test`, on `bun:test`. Two styles: a `QueryClient` whose `invalidateQueries` is replaced by a
recorder, to assert filters and options; and a real `QueryClient` seeded with `setQueryData`, to
assert that the prefix invalidates every `metrics` duration and leaves `['config']` alone.
