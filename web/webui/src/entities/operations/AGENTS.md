# entities/operations — how this module hangs together

The operations log of the web UI — one row per delivery of one table row to one sink — as three
FSD **entity** slices: `filter` (what to show), `list` (the paged, live table) and `details` (the
drawer for one row). There is **no top-level `index.ts`**: each slice is imported on its own,
`@/entities/operations/filter`, `/list`, `/details`. The only consumer is `pages/operations`, which
lays the three out and owns everything URL-related.

## Layout

| Path | Role |
|---|---|
| `filter/index.ts` | `OperationsFilter`, `useOperationsFilter`, `defaultFilterValues`, `operationItems`, `statusItems`, types `OperationsFilterState` / `OperationsFilterValues` |
| `filter/model/store.ts` | the persisted zustand filter store |
| `filter/model/options.ts` | the fixed operation and status option lists |
| `filter/ui/operations-filter.tsx` | five `MultiComboBox` facets and the search input |
| `list/index.ts` | `OperationsList` only — the hook, the row type and the mappers stay private |
| `list/model/use-operations.ts` | block loading, the debounced search, the live refetch |
| `list/model/blocks.ts` | pure block arithmetic: `BLOCK_SIZE`, `blocksForRange`, `clampBlocks`, `sameBlocks`, `BlockPage`, `pickTotal`, `rowAt` |
| `list/model/to-request.ts` | filter store → request filter, which is also the query key |
| `list/model/map-operation.ts`, `types.ts` | API record → `Operation` / `Attempt` |
| `list/ui/` | the `DataTable` wiring, its columns, the expandable attempts table |
| `details/index.ts` | `OperationDetails`, `useOperationDetailsStore` |
| `details/model/` | the selection store; `toHeaders` over a record's metadata |
| `details/ui/` | the sheet shell, a desktop panel body and a mobile tabs body built from shared rows |
| `*/@x/*.ts` | the only doors between slices — see below |

There are no stories in this module.

## Imports

The slices need each other, and every such edge goes through an `@x` file, never into another
slice's `model/` or `index.ts`:

| Door | Opens to | Exposes |
|---|---|---|
| `filter/@x/list.ts` | `list` | `useOperationsFilter`, `OperationsFilterState`, the two option lists |
| `list/@x/details.ts` | `details` | the `Operation` type, nothing at runtime |
| `details/@x/list.ts` | `list` | `useOperationDetailsStore` |

`list` and `details` therefore point at each other: the list writes the selection, the drawer
renders the list's row type. Outwards, only `filter/ui` crosses entities — `useConfig`
(`entities/config/config/@x/operations`), `useLiveControlsChartStep`
(`entities/config/state/@x/operations`) and `useMetrics` (`entities/metrics/@x/operations`), all to
build option lists. From `shared`: `api` (`Operations.list` and the generated models), `live`
(`useLiveOperations`), `state` (`keys`, `OperationsFilterKey`), `ui/*` and `utils`.

## The filter store

`useOperationsFilter` persists its six values to localStorage under `operations:filter`
(`version: 1`; `partialize` keeps the setters out). A facet has three states, and they are not
interchangeable:

- **`null`** — `sinks`, `targets`, `tables` only: everything is allowed. Their options arrive late
  (table and sink types from the config, targets from the metrics topology), so "all" cannot be
  stored as a list. `orAll` in the filter UI folds a selection that covers every option back to
  `null`, which keeps options that load later allowed.
- **a list** — narrowed. `operations` and `statuses` have fixed options and default to all of them.
- **`[]`** — matches nothing. `toOperationsFilter` returns `undefined` for it, the block queries are
  disabled, and the list reports `rowCount: 0` without a request.

The target options are read from the metrics query at the chart step the rest of the UI already
polls, so the filter adds no metrics request of its own.

## URL state belongs to the page

Nothing here reads the router. `pages/operations/model/search.ts` defines the search params
(`table`, `sink`, `target`, `operation`, `result` — also the route's `validateSearch`) and
`toFilterValues`; `use-search-filter.ts` calls the store's `apply` with the result and then replaces
the history entry with an empty search. So a link into `/operations` is a one-shot **preset** of the
persisted filter, not a second source of truth: facets it leaves out reset to all, the search text
clears, and after that the store alone decides. `apply`, `defaultFilterValues` and the option lists
are exported for exactly that code.

## Loading: blocks, not an infinite query

`useOperations` reads `POST /api/operations` in blocks of `BLOCK_SIZE` (100), one cached query per
block under `keys.queries.operations(filter, block)`; `fetchBlock` rebuilds the request body from
its key (`from: block * 100`, `to: (block + 1) * 100`). `from` / `to` are indices into the matches,
newest first — see `internal/application/webui/get_operations/AGENTS.md`.

- `DataTable` renders `rowCount` rows by index and never fetches; it reports the visible range and
  asks `getRow(index)`, which answers `undefined` (a skeleton) until that block is loaded.
- **Block 0 is always requested**, whatever the viewport: every response carries the matching
  total, and block 0 keeps it known. It alone uses `keepPreviousData`, so a filter change keeps the
  old rows on screen until the new ones arrive.
- The other blocks are clamped by block 0's total (`clampBlocks`), so nothing past the end is
  requested. `rowCount` is the total of the most recently fetched page (`pickTotal`, by the
  page's `updatedAt`, which `toPage` copies from `receivedAt`, the `Date.now()` that `fetchBlock`
  adds to the response), not block 0's.
- The backend answers with the **clamped** window; `rowAt` ignores a page whose `from` is not its
  block's start rather than show rows at the wrong index.
- The request filter **is** the query key, hence `toOperationsFilter` sorts every list and leaves
  out a facet that allows everything. The search text is debounced 300 ms before it reaches the key.
- `toPage` and `combinePages` are module-level on purpose: stable references are what let TanStack
  Query memoize `select` / `combine`.

`map-operation.ts` is the vocabulary boundary: API `success` / `failure` become `delivered` /
`failed` (`to-request.ts` maps them back for the filter), and an attempt with any status but
`success` — `retry` included — is `failed`. `attempts` falls back to `retries + 1` when the record
has no attempt list; `retries` is set only when there was more than one attempt, which is what
makes a row expandable. int64 fields arrive as `bigint` although the generated types say `number`,
hence `toNumber`.

## Live updates

The `/api/live` batches say only where a delivery went
(`internal/application/webui/stream_updates/AGENTS.md`), so they are never merged into the cache.
`useLiveRefetch` subscribes through `useLiveOperations` and invalidates `keys.prefixes.operations()`
after 300 ms of quiet, at most 1 s into a steady stream, with `cancelRefetch: false` so a slow
request is not restarted by the next batch.

- The list refreshes **itself, and only while mounted**. `shared/live`'s `LiveUpdates` invalidates
  metrics on every batch but touches operations only on reconnect
  (`shared/state/invalidate.ts`). That filter has no `refetchType`, so TanStack's default
  `'active'` applies: a mounted list is refetched, inactive operations queries are only marked
  stale.
- The prefix is `['operations']`, which also covers `keys.queries.recentOperations` — the flows
  inspector's list. A key added under that root is invalidated from here too.
- Rows are positions counted from the newest, so every delivery shifts them; a refetch replaces the
  loaded blocks wholesale. `BLOCK_STALE_TIME_MS` (5 s) only matters for blocks scrolled back to.

The log itself is a bounded ring that records only while a live client is connected:
`internal/adapter/operations_collector/AGENTS.md`.

## Details

`useOperationDetailsStore` keeps the **whole selected row**, not an id: a finished delivery never
changes and there is no endpoint to fetch one by id. The drawer therefore keeps showing a row after
a refetch has moved or dropped it from the list. It is not persisted.

`OperationDetails` is a non-modal `Sheet` — no overlay, outside interaction not dismissing it — so
the rows beneath stay clickable. It is positioned `absolute` on desktop and expects the positioned,
clipped grid region `pages/operations` renders it in; on mobile it portals to the viewport as a
bottom sheet with a tabs body instead of the panel. `shown` holds the last selection through the
close animation, when `selected` is already `null`.

`toHeaders` lists every metadata entry and strips the `header.` prefix the sinks file headers
under, so headers and other metadata (the HTTP sink's `method` and `status`) share one list.

## What changes together

- An operation kind or a delivery status: `filter/model/options.ts`, the `Operation` unions in
  `list/model/types.ts`, `API_STATUSES` in `to-request.ts`, `map-operation.ts` — and the backend
  `enum` tags. `pages/operations/model/search.ts` validates URL values against the option lists.
- A field of `OperationsFilterValues`: the store's `partialize` (and its `version`),
  `OperationsFilterInput` / `toOperationsFilter`, the filter UI, the page's `toFilterValues`.
- `detailHeight` in `operations-list.tsx` (`62 + 44` per attempt) mirrors the header and row heights
  of `operation-retries.tsx`.
- The drawer's desktop width (`w-95`) and the page's `pr-99` on the grid region.

## Tests

`bun:test`, next to the code: `blocks.test.ts` (block arithmetic, the clamped-page case),
`to-request.test.ts` (omitted, empty and sorted facets, the trimmed search), `map-operation.test.ts`
(bigint conversion, attempts and the retries fallback), `headers.test.ts`. The hooks, the stores and
the components have none.
