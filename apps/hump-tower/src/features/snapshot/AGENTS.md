# features/snapshot — how this module hangs together

The Snapshot page body: pick tables and row filters, start a run, watch it, read the result. The
only slice of the `features` layer. It is a client of the backend snapshot runner and holds no run
state of its own: the server's state machine (`idle` → `processing` → `done`) decides which view is
on screen.

- **Public contract**: `index.ts` exports `Snapshot` only. Consumer: `pages/snapshot`.
- **Not `shared/live`**: the header badge (`entities/config/state/ui/snapshot-status.tsx`) reads
  `useLiveSnapshot()` from `/api/live`: state plus a server-computed percent. This slice opens its
  own stream to `GET /api/snapshot`, the only source of per-table detail. The two share neither a
  cache nor a connection; the percent here is computed client-side by `aggregate`.

`ui/idle`, `ui/processing` and `ui/done` hold one view each; each folder's `index.tsx` exports
`<State>View` and re-exports `<State>ActionBar`.

## Where state comes from

| State | Source | Owner |
|---|---|---|
| which view, run tables, counters, timestamps | SSE `GET /api/snapshot` | `model/snapshot.ts` |
| configured tables, columns, suggested filter, sinks | `POST /api/snapshot/tables` | `useAvailableTables` |
| row estimate per selected table | SSE `POST /api/snapshot/estimate` | `useTablesEstimate` |
| selection, per-table `where`, name filter | zustand store, `localStorage` key `snapshot-tables` | `useTables` |
| the filter text while it is being typed | `useState` in `FilterCell` | `ui/idle/tables.columns.tsx` |

## The run stream

Backend: `apps/hump/internal/application/hump_tower/stream_snapshot/AGENTS.md`. Every event is the
**whole** state, so the `streamedQuery` reducer is `(_, event) => event`: the last event replaces
the cache. Do not accumulate.

- All readers go through one `queryOptions` under `keys.queries.snapshot()`, so they share one
  connection.
- The generated client gets `sseMaxRetryAttempts: 1`; reconnecting is react-query's job. The
  `streamFn` **throws when the stream ends without an abort**, which turns a closed stream into a
  failed query, and `retry: true` reopens it with backoff capped at `MAX_RETRY_DELAY_MS`.
- `select` is the single place wire models become view types. It switches on `event.state`, the
  single-value enum the backend keeps on each payload so the client can tell the three shapes
  apart (the flattened OpenAPI spec drops event names).
- A done table has no `where` and no `estimatedRows`, so `select` sets `Table.where` to `''` and
  `Table.totalRows` to `0` there; only done tables get `error`. Do not read `totalRows`, `percent`
  or `where` in `ui/done`.
- Run tables are identified as `` `${schema}.${name}` ``; the tables endpoint calls the same field
  `table`, and `useAvailableTables` builds the same id from it.

## State → view

`ui/snapshot.tsx` keeps two `Record<SnapshotState['state'], ReactNode>` maps, `views` and
`actionBars`, of prop-less elements. `Snapshot` renders `null` until the first event arrives.

The children of `ProcessingView` and `DoneView`, plus `DoneActionBar`, read the state from
`SwitchViewValueContext` via
`useSwitchViewValue<SnapshotState>()` and start with `if (data?.state !== '<own>') return null`.
This is what the cross-fade needs: on a state change `useSwitchAnimation` keeps the previous view
mounted until its exit animation ends, and `SnapshotStates` remembers the last state object per
state name (`shown`) to feed the leaving view. So a processing view fades out over its final
numbers rather than over a `done` payload it cannot render. A new component under `ui/processing`
or `ui/done` must read the context the same way, not call `useSnapshotState()`. The idle view
ignores the context; everything it shows comes from `model/tables.ts`.

Table statuses follow the backend vocabulary (`apps/hump/internal/adapter/snapshot/AGENTS.md`,
"Statuses"): `wait`/`processing`/`done` while running, `success`/`error` once finished.
`TableStatus` is the union of both. A table that failed mid-run arrives as plain `done` with no
error, so the processing view shows it as healthy; the failure only appears in the done view.

## Start and stop

Backend: `apps/hump/internal/application/hump_tower/start_snapshot/AGENTS.md` (202, no body) and
`apps/hump/internal/application/hump_tower/stop_snapshot/AGENTS.md` (204, no body). Both mutations only await the call: no
`onSuccess`, no invalidation, no cache write. The UI changes when the stream delivers the next
state; never switch views optimistically from a mutation.

- `useStopSnapshot` is one call with two meanings, as on the backend: `ProcessingActionBar` stops
  a run, `DoneActionBar` clears a finished one back to `idle` ("Run another snapshot").
- Every failure (409 already running / not running, 422 invalid `where`) is a toast with
  `errorMessage(error)`. Nothing is rendered inline for a failed start.

## The start form

Backend: `apps/hump/internal/application/hump_tower/get_snapshot_tables/AGENTS.md` and
`apps/hump/internal/application/hump_tower/get_snapshot_estimates/AGENTS.md`.

- **The effective filter is `wheres[id] ?? table.suggestedWhere`**, spelled out in three places:
  `useTablesEstimate`, `ui/idle/tables.tsx` and `ui/idle/action-bar.tsx`. They must agree: the
  estimate is keyed by it and the start request sends it. `??` is deliberate: a filter the user
  cleared is stored as `''` and stays empty rather than falling back to the suggestion.
- The store persists `selectedIds` (as an array) and `wheres`; `filter` is not persisted. Persisted
  ids of tables that are no longer returned are ignored, because `useSelectedTables` filters the
  fetched list by the set.
- `FilterCell` commits on **blur**, not per keystroke: `setWhere` changes the estimate's query key
  (`keys.queries.snapshotEstimate(schema, table, where)`), which starts a new count.
- One estimate request per selected table, each with a single-table body, so events need no
  matching by name. The stream yields the rough count, then the exact one replaces it; `fetching`
  (the pulse) stays on while the stream is open. An `error` event is thrown, a stream with no count
  throws `No estimate received`, and `retry` is off.
- **This is how a bad `where` surfaces.** The estimate's error is shown under the input and marks
  `WhereInput` `invalid`; `IdleActionBar` lists the failing tables and disables "Run snapshot". Run
  is also disabled while any selected table is still estimating. The 422 from `start_snapshot` is
  a backstop that ends in a toast. Any estimate failure is labelled a "filter error", whatever
  caused it.
- "Select all" toggles only the tables visible under the name filter (`toggleAll(ids)`).
- The sinks listed here are not filtered by operation, while a run lists only sinks that take
  `insert` (`get_snapshot_tables/AGENTS.md`). The idle and done sink lists can differ.

## When the backend contract changes

After regenerating the client (`apps/hump-tower/AGENTS.md`, Commands):

- **A field of a `processing`/`done` payload**: the matching `case` in `select`
  (`model/snapshot.ts`) and `Table`/`Sink` in `model/types.ts`. The two cases map tables
  separately; keep both in step.
- **A new or renamed run state**: a `case` in `select`, a member of `SnapshotState`, and an entry
  in both `views` and `actionBars` (the compiler asks for them). The `/api/live` badge has its own
  state enum (`SnapshotEventModel`) and is updated separately.
- **A table status**: `STATUS_BADGE` in `ui/processing/tables.columns.tsx` (compiler-checked) and
  the hand-written legend in `ui/processing/tables.tsx`; `ui/done` compares against `'error'`
  (`hasFailed`, `failedCount`).
- **An estimate event**: the `'error' in event` / `'count' in event` checks in
  `useTablesEstimate`: payloads are told apart by shape, not by name.

## Things that will bite

- Two different `snapshotQuery`: the exported SQL-text helper in `model/selectors.ts` and the
  private query options constant in `model/snapshot.ts`.
- `aggregate().percent` is `0` when `totalRows` is `0` and is capped at 100.
- `DoneSinks` groups by `type` + `host`, summing a sink's targets together; the per-table column
  in `ui/done/tables.columns.tsx` keeps targets apart.
