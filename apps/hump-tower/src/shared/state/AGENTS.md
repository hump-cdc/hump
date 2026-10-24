# shared/state — how this module hangs together

The TanStack Query key registry (`keys.ts`) and the two invalidation routines the live stream
runs (`invalidate.ts`). No store, no hooks, no React: plain functions over a `QueryClient`. Its
only import from the app is type-only, from `@/shared/api`.

## The key scheme

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

`shared/live/live-updates.tsx` is the only caller: `invalidateForLiveEvent` on a non-empty
`operations` batch, `invalidateForLiveReconnect` when the stream starts a second or later attempt.
The filter lists are `LIVE_EVENT_QUERIES` and `LIVE_RECONNECT_QUERIES`.

- A live event says only that deliveries happened (see
  `apps/hump/internal/application/hump_tower/stream_updates/AGENTS.md`); the numbers come from
  refetching.
- On an event only mounted metrics queries refetch (the react-query default). After a reconnect
  events were missed, so metrics refetch **even while unmounted** (`refetchType: 'all'`);
  operations are only marked stale.
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

`keys.test.ts` compares the built keys with literal tuples. `invalidate.test.ts` has two styles:
a `QueryClient` whose `invalidateQueries` is replaced by a recorder, to assert filters and
options; and a real `QueryClient` seeded with `setQueryData`, to assert that the prefix
invalidates every `metrics` duration and leaves `['config']` alone.
