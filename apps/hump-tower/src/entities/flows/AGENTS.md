# apps/hump-tower/src/entities/flows — how this package hangs together

FSD `entities` layer: everything on the Flows page. Three slices, each imported by its own path
(no top-level `index.ts`):

| Slice | `index.ts` exports | Role |
|---|---|---|
| `@/entities/flows/flows` | `Flows` | the topology canvas (tables → sinks), summary and detailed views |
| `@/entities/flows/toolbar` | `FlowsToolbar` | the Tables and Sinks multi-selects |
| `@/entities/flows/inspector` | `FlowInspector`, `useFlowInspectorStore` | the drawer for the clicked node or edge |

The only consumer is `pages/flows`, which passes `FlowsToolbar` and `FlowInspector` to `Flows` as
`overlay` and builds `viewportInsets` from them (`pages/flows/AGENTS.md`).

Slices reach each other through `@x` in one direction, `toolbar → flows → inspector`. From other
entities: `useMetrics` and topology types (`entities/metrics`), `useLiveControlsChartStep`
(`entities/config/state`), `useConfig` (`entities/config/config`, inspector only).

## Data sources

| Endpoint | Query key | Read by | Backend use case |
|---|---|---|---|
| `POST /api/metrics` | `queries.metrics(duration)` | `useFlowsTopology`, `useInspectorTopology` | `get_metrics` |
| `POST /api/operations` | `queries.recentOperations(filter)` | `useRecentOperations` | `get_operations` |
| `GET /api/config` | from `getApiConfigOptions()` | `EdgeDetails`, via `useConfig` | `get_config` |
| `GET /api/live` | — | `useLiveOperations` | `stream_updates` |

- Canvas, toolbar and inspector read the metrics at the same chart step, so they share one query.
  The step (`1m` / `5m` / `10m`) sets the window the p95 figures cover.
- This module never refreshes the topology itself; refresh is owned by `entities/metrics`.
- Its own live subscriptions do two things only: flash edges, and refetch the recent operations,
  debounced 300 ms with a 1 s max wait so a steady stream cannot defer it forever.

## Kit and entity

`shared/ui/flow`, `flow-summary` and `flow-detailed` own the canvas: column layout (callers give
no coordinates), selection highlight, revealing the selection, and `ref.animate`. This module
supplies the data and the meaning: which nodes and edges exist, how heavy an edge is, what a click
selects.

`TableSpec`, `SinkSpec` and `DetailedSinkSpec` (`flows/model/types.ts`) are `ComponentProps` of
the kit's node components, and `toFlowsTopology` builds exactly those, so a changed kit prop shows
up as a type error in `topology.ts`.

Edge weight is the kit's `style` contract: opacity 0.25–1 and stroke width 1.5–5, scaled by the
edge's share of the busiest edge **of its own view**. Summary edges sum the connections per
table → sink; detailed edges are the connections as reported.

## Ids are the contract with the live stream

- table: `schema.table` (`tableId`), which is also the metrics table name;
- sink: `type|host` (`sinkId`): a sink type alone does not identify an endpoint;
- target row: the target string, unique only within its sink; as a selection it is the kit's
  `sinkTargetKey(sinkId, target)`;
- edges: `table>sink` in summary, `table>sink>target` in detailed.

A live operation becomes `animate(tableId(schema, table), sinkId(sink, host), target)`, and the
kit finds the edge by those ids. A mismatch is a silent no-op, not an error; `topology.test.ts`
("ids match the fields of a live operation") pins it.

Edge and target ids differ between the views, so `FlowsModeSwitch` clears the selection before it
changes the mode.

## Selection carries identity, not figures

`SelectedItem` holds ids, labels and the sink key (`type`, `host`). The details components resolve
the figures from the latest metrics on every render, so an open inspector stays current, and
`resolve*` returns `undefined` for an item the metrics no longer report.

- The success share is floored to a whole percent (one failure never reads as 100%); failures are
  derived from it. The canvas marks a sink or target as failing below a 0.5 share.
- Edge figures sum `connections`; a summary edge (no `target`) takes every target of the route.
  An edge has no p95.
- `resolveEdgeOperations` returns `undefined` when the edge receives all four operations, so the
  "Operations" card appears only when something is filtered out.
- `toRecentFilter` narrows by `hosts`; `toOperationsSearch` cannot: the `/operations` search
  (`pages/operations/model/search.ts`) has no host facet, so "View in Operations" narrows a sink
  by type only.

## Gotchas

- Persisted to localStorage: `hump.flows-filter`, `hump.flows-mode` (default `detailed`). The
  selection is not persisted.
- Both stores are at `version: 1` with a `migrate` (`migrateFlowsFilter`, `migrateTopologyMode`)
  that falls back to the default for an invalid stored value. zustand discards stored state on a
  version mismatch when there is no `migrate`, so bumping a version means extending that function.
- A filter value is `'all'` or a list of ids. The toolbar stores `'all'` once every option is
  picked, so tables and sinks that appear later stay visible; a stored list hides them.
- `filterFlowsTopology` must narrow both `sinks` / `edges` and `detailedSinks` / `detailedEdges`.
- `FlowInspector` keeps the last selection in `shown` to render through the close animation. It is
  non-modal and unportalled (`portal={false}`), so it docks inside the canvas region `Flows`
  renders `overlay` into. `RecentOperations` is rendered on desktop only.
- The desktop sheet is `w-75`, and `pages/flows` hardcodes the same width twice; change all three
  together (`pages/flows/AGENTS.md`).
