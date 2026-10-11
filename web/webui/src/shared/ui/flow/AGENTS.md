# Flow

The read-only React Flow (`@xyflow/react`) canvas and the building blocks the topology views are
made of. It has no story and is not rendered directly: pick `FlowSummary` (`@/shared/ui/flow-summary`)
or `FlowDetailed` (`@/shared/ui/flow-detailed`), which wrap it.

Consumers:

- `shared/ui/flow-summary`, `shared/ui/flow-detailed` — the canvas, node views, hooks, constants.
- `entities/flows/flows`, `pages/flows` — the `ViewportInsets` type.
- `entities/flows/flows/ui/flows.mode.tsx` — `FlowControlButton`.

## Exports

Everything is re-exported from `flow.tsx` through `index.ts`.

| Group | Export | What it is |
|---|---|---|
| Canvas | `Flow` | the canvas component, see Props |
| | `FlowControlButton` | xyflow's `ControlButton`, for the `controls` slot |
| Node views | `TableNodeView` | renderer of a `table` node: name, total, four operation cells, a source handle on the right |
| | `LabelNodeView` | renderer of a `label` node: a non-interactive column caption |
| Config elements | `TableNode` | `<TableNode id name ops />`, renders nothing, carries `flowNodeType: 'table'` |
| Node parts | `MetricHint` | wraps a metric element in a tooltip |
| | `SinkIcon`, `SinkIconTile` | sink-type logo, bare or inside a square tile |
| Hooks | `useFlowSelection` | flags the selected node/edge, marks animated edges, optional edge `zIndex` |
| | `useSelectionHandlers` | node/edge/pane click handlers → selection callbacks |
| | `useEdgeAnimation` | `animatedEdges` map plus `animateEdge(id, durationMs?)` |
| | `useMeasuredNodes` | keeps measured node sizes across rebuilt `nodes` |
| Helpers | `collectNodeSpecs` | splits children into config elements (`FlowNodeSpec[]`) and the rest |
| | `columnLabel` | a `label` node `LABEL_OFFSET_Y` above a column's first node |
| Constants | `TABLE_COLUMN_X` (0), `SINK_COLUMN_X` (584), `LABEL_OFFSET_Y` (36), `DEFAULT_ANIMATE_DURATION` (1300 ms) | layout geometry and the default edge animation time |
| Types | `FlowNodeBase`, `FlowNodeConfig`, `FlowNodeSpec`, `TableNodeData`, `ViewportInsets`, `TableNodeSelectHandler`, `EdgeSelectHandler`, `ClearSelectionHandler` | |

## Usage

A view built on `Flow` declares its nodes as config-element children, lays them out itself and
passes ready nodes down (`flow-summary/flow-summary.tsx`):

```tsx
import { Flow, useFlowSelection, useSelectionHandlers } from '@/shared/ui/flow'

const { nodes, rest } = useMemo(() => partitionChildren(children), [children]) // collectNodeSpecs + layout
const { selectedNodes, selectedEdges } = useFlowSelection(nodes, edges, selected, animatedEdges)
const { handleNodeClick, handleEdgeClick, handlePaneClick } = useSelectionHandlers({ selected, ...callbacks })

<Flow
    nodes={selectedNodes}
    edges={selectedEdges}
    nodeTypes={nodeTypes}
    onNodeClick={handleNodeClick}
    onEdgeClick={handleEdgeClick}
    onPaneClick={handlePaneClick}
    revealId={selected}
>
    {rest}
</Flow>
```

An extra control button (`entities/flows/flows/ui/flows.mode.tsx`):

```tsx
<FlowControlButton title="Summary" aria-label="Summary" aria-pressed={active} className="flow-controls-joined">
    <Icon />
</FlowControlButton>
```

Constraints:

- `Flow` is `h-full w-full`: the parent must have a height.
- `Flow` mounts its own `ReactFlowProvider`; children render inside `<ReactFlow>` as overlays.
- `nodes`, `edges`, `onNodesChange`, `onMoveStart` are wrapped (transition, measured sizes, user-move
  tracking) and `colorMode` is fixed to `light`; these are set after the props spread. The defaults
  `nodesDraggable={false}`, `nodesConnectable={false}`, `nodesFocusable`, `edgesFocusable`, `fitView`,
  `fitViewOptions={{ maxZoom: 1 }}`, `maxZoom={1}`, `proOptions` come before it and can be overridden.
- `MetricHint` uses `@/shared/ui/tooltip`, so every node view needs a `TooltipProvider` above the
  canvas; its child must forward props to a DOM node (`TooltipTrigger asChild`).
- `useSelectionHandlers` selects only nodes of type `table` and `sink`; a click on the selected item
  or on the empty pane calls `onClearSelection`.
- A layout transition (400 ms) starts only when node ids, node positions or edge ids change; the first
  render is not animated.

## Props

`Flow` — plus every `ReactFlow` prop.

| Prop | Type | Default | Description |
|---|---|---|---|
| `revealId` | `string` | — | id of a node or edge to pan into view whenever it changes |
| `revealKey` | `string` | `revealId` | selection behind `revealId`; a change re-reveals even if `revealId` stays the same |
| `viewportInsets` | `ViewportInsets` (`{ top, right, bottom, left }`, px) | all `0` | canvas margins covered by floating overlays; kept clear by reveal and fit view |
| `controls` | `ReactNode` | — | extra `FlowControlButton`s, rendered above the zoom buttons in the same card |

`TableNode` (config element; the same fields are `TableNodeData` plus `id`).

| Prop | Type | Default | Description |
|---|---|---|---|
| `id` | `string` | required | node id, referenced by edges and `selected` |
| `name` | `string` | required | table name |
| `ops` | `Record<'insert' \| 'update' \| 'delete' \| 'truncate', { count: number; p95?: number }>` | required | count and p95 latency (ms) per operation |

`MetricHint`

| Prop | Type | Default | Description |
|---|---|---|---|
| `hint` | `string` | required | tooltip text |
| `children` | `ReactElement` | required | the metric element; receives the trigger props |

`SinkIcon` takes `type: string`; `SinkIconTile` takes `type: string` and `className?: string`.

## Semantic styles

**Canvas** (`flow.css` maps xyflow's `--xy-*` variables to the tokens, so light/dark follow `.dark`):

| Part | Token |
|---|---|
| background / dots | `--canvas` / `--hairline-strong` |
| edge, selected edge | `--ink-muted` (width 2), `--primary` |
| handle | `--canvas` fill, `--hairline-strong` border; `--primary` border on a selected node or a `.selected-target` handle |
| controls button | `--surface-2`, hover `--surface-3`; icon `--ink-subtle`, hover `--ink`; `--hairline` borders |
| controls button `[aria-pressed="true"]` | `--primary` icon: the active one of a toggle group |
| `.flow-controls-joined` | no divider below: buttons of one group sit flush |

Zoom in / zoom out are disabled at the max / min zoom.

**Table node** — cva `selected`, on a `bg-surface-1` card:

| `selected` | Classes |
|---|---|
| `true` | `border-primary shadow-glow shadow-primary/40`, header icon `text-primary` |
| `false` (default) | `border-hairline`, hover `border-hairline-strong shadow-glow shadow-ink-subtle/30`, header icon `text-ink-subtle` |

Operation cells (label `text-ink-tertiary`, p95 `text-ink-subtle`), count tone per operation:

| Operation | Label | Count tone |
|---|---|---|
| `insert` | `INS` | `text-success` |
| `update` | `UPD` | `text-ink` |
| `delete` | `DEL` | `text-danger` |
| `truncate` | `TRN` | `text-ink-muted` |

**Label node** — `text-ink-tertiary text-xs tracking-caps`, `pointer-events-none`.

**Sink icon** — `type` → logo: `kafka`, `nats-core` / `nats-jetstream`, `http`, `rabbitmq` /
`rabbitmq-stream`, plus the ghost types `clickhouse`, `s3`, `snowflake`; any other type falls back to
`Plug`. The tile is `bg-surface-2 border-hairline text-primary`.
