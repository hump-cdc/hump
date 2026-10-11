# web/webui/src/entities/flows — how this package hangs together

FSD `entities` layer: everything on the Flows page — the topology canvas (tables → sinks), the
toolbar that filters it and the inspector drawer for the clicked node or edge. Three slices,
`flows`, `inspector` and `toolbar`, and **no top-level `index.ts`**: each slice is imported by its
own path. The only consumer is `src/pages/flows/ui` (`flows.tsx`, `use-insets.ts`).

## Layout

| File | Role |
|---|---|
| `flows/model/topology.ts` | `toFlowsTopology`: metrics topology → specs of both views; ids |
| `flows/model/use-flows-topology.ts` | the metrics query, mapped; empty until it answers |
| `flows/model/filter.ts` | persisted table / sink selection and `filterFlowsTopology` |
| `flows/model/mode.ts` | persisted `summary` / `detailed` mode (default `detailed`) |
| `flows/model/types.ts` | spec types, derived from the kit components' props |
| `flows/ui/flows.tsx` | `Flows`: filters, cross-fades the two views, hosts the page's `overlay` |
| `flows/ui/flows.summary.tsx`, `flows.detailed.tsx` | a view: kit canvas, selection, live flashes |
| `flows/ui/flows.mode.tsx` | the mode toggle passed to the canvas as `controls` |
| `inspector/model/store.ts`, `types.ts` | `useFlowInspectorStore` and `SelectedItem`, the selection |
| `inspector/model/resolve.ts` | figures of the selection, from the metrics and the config |
| `inspector/model/use-inspector-topology.ts` | the metrics query the canvas reads |
| `inspector/model/recent-filter.ts`, `use-recent-operations.ts` | its five newest deliveries |
| `inspector/model/operations-search.ts` | `/operations` search params for "View in Operations" |
| `inspector/ui/flow-inspector.tsx` | `FlowInspector`: the non-modal sheet |
| `inspector/ui/inspector-details.tsx`, `details.*.tsx` | one component per selection kind |
| `inspector/ui/recent-operations.tsx` | the recent deliveries list |
| `toolbar/ui/flows-toolbar.tsx` | `FlowsToolbar`: the Tables and Sinks multi-selects |

Tests (`bun:test`) sit next to `topology`, `resolve`, `recent-filter` and `operations-search`.
There are no stories here; the canvas stories belong to the kit.

## Public contract and imports

| Slice | `index.ts` exports |
|---|---|
| `@/entities/flows/flows` | `Flows` |
| `@/entities/flows/inspector` | `FlowInspector`, `useFlowInspectorStore`, type `SelectedItem` |
| `@/entities/flows/toolbar` | `FlowsToolbar` |

Slices reach each other only through `@x`, in one direction, `toolbar → flows → inspector`:

- `flows/@x/toolbar.ts` — `useFlowsTopology`, `useFlowsFilterStore`, `FlowsFilterSelection`;
- `inspector/@x/flows.ts` — `useFlowInspectorStore`.

From other entities, also through their `@x` files:

- `entities/metrics/@x/flows.ts`, `@x/inspector.ts` — `useMetrics` and the topology types;
- `entities/config/state/@x/flows.ts`, `@x/inspector.ts` — `useLiveControlsChartStep`;
- `entities/config/config/@x/inspector.ts` — `useConfig`.

The page composes the three: it passes `FlowsToolbar` and `FlowInspector` to `Flows` as `overlay`
and builds `viewportInsets` from them, so a revealed selection is not hidden under either.

## Data sources

Backend guides are `internal/application/webui/<use case>/AGENTS.md`; query keys come from `keys`
in `shared/state`.

| Endpoint | Query key | Read by | Use case |
|---|---|---|---|
| `POST /api/metrics` | `queries.metrics(duration)` | `useFlowsTopology`, `useInspectorTopology` | `get_metrics` |
| `POST /api/operations` | `queries.recentOperations(filter)` | `useRecentOperations` | `get_operations` |
| `GET /api/config` | from `getApiConfigOptions()` | `EdgeDetails`, via `useConfig` | `get_config` |
| `GET /api/live` | — | `useLiveOperations` | `stream_updates` |

- Canvas, toolbar and inspector call `useMetrics` with the same chart step, so they share one
  query. The step is the `1m` / `5m` / `10m` live control; it sets the window the p95 figures cover.
- This module never refreshes the topology itself. The metrics query repolls every minute, and
  `LiveUpdates` (`shared/live`) invalidates `keys.prefixes.metrics()` on every live batch.
- The module's own live subscriptions do two things only: flash edges, and refetch the recent
  operations — debounced 300 ms with a 1 s max wait, so a steady stream cannot defer it forever.
- `shared/prometheus` is not used here.

## Kit and entity

`shared/ui/flow`, `flow-summary` and `flow-detailed` own the canvas: column layout (callers give no
coordinates), selection highlight, revealing the selection, and `ref.animate`. This module supplies
the data and the meaning: which nodes and edges exist, how heavy an edge is, what a click selects.
`shared/ui/sink` is not imported here.

`TableSpec`, `SinkSpec` and `DetailedSinkSpec` are `ComponentProps` of the kit's node components,
and `toFlowsTopology` builds exactly those, so the views spread them (`<FlowSummarySink {...sink} />`).
A changed kit prop therefore shows up as a type error in `topology.ts`.

Edge weight is the kit's `style` contract: opacity 0.25–1 and stroke width 1.5–5, scaled by the
edge's share of the busiest edge **of its own view**. Summary edges sum the connections per
table → sink; detailed edges are the connections as reported, with `targetHandle` = the target.

## Ids are the contract with the live stream

- table: `schema.table` (`tableId`), which is also the metrics table name;
- sink: `type|host` (`sinkId`) — a sink type alone does not identify an endpoint;
- target row: the target string, unique only within its sink; as a selection it is the kit's
  `sinkTargetKey(sinkId, target)`;
- edges: `table>sink` in summary, `table>sink>target` in detailed.

A live operation is turned into `animate(tableId(schema, table), sinkId(sink, host), target)` and
the kit finds the edge by those ids. A mismatch is a silent no-op, not an error —
`topology.test.ts` ("ids match the fields of a live operation") pins it.

Because edge and target ids differ between the views, `FlowsModeSwitch` clears the selection
before it changes the mode.

## Selection carries identity, not figures

`SelectedItem` holds ids, labels and the sink key (`type`, `host`). The details components resolve
the figures from the latest metrics on every render, so an open inspector stays current, and
`resolve*` returns `undefined` for an item the metrics no longer report.

- Sink and target figures come from the sink entry: the success share is floored to a whole
  percent (one failure never reads as 100%), failures are derived from it. The canvas marks a
  sink or target as failing below a 0.5 share.
- Edge figures sum `connections`; a summary edge (no `target`) takes every target of the route.
  An edge has no p95.
- `resolveEdgeOperations` reads the config and returns `undefined` when the edge receives all
  four operations, so the "Operations" card appears only when something is filtered out.
- `toRecentFilter` narrows by `hosts`; `toOperationsSearch` cannot — the `/operations` search
  (`pages/operations/model/search.ts`) has no host facet, so the link narrows a sink by type only.

## Gotchas

- Persisted to localStorage: `hump.flows-filter`, `hump.flows-mode`. The selection is not persisted.
- A filter value is `'all'` or a list of ids. The toolbar stores `'all'` once every option is
  picked, so tables and sinks that appear later stay visible; a stored list hides them.
- `filterFlowsTopology` must narrow both `sinks` / `edges` and `detailedSinks` / `detailedEdges`.
- `FlowInspector` keeps the last selection in `shown` to render through the sheet's close
  animation. It is non-modal and unportalled (`portal={false}`), so it docks inside the canvas
  region `Flows` renders `overlay` into.
- The desktop sheet is `w-75`; `pages/flows/ui/use-insets.ts` hardcodes the same 300 px as
  `INSPECTOR_WIDTH` instead of measuring it. Change them together.
- `RecentOperations` is rendered on desktop only.
