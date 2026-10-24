# Flow

The read-only React Flow (`@xyflow/react`) canvas and the parts shared by the topology views. It has
no story and is not rendered directly: use `FlowSummary` (`@/shared/ui/flow-summary`) or
`FlowDetailed` (`@/shared/ui/flow-detailed`), which wrap it. Outside those two, only the
`ViewportInsets` type (`entities/flows/flows`, `pages/flows`) and `FlowControlButton`
(`entities/flows/flows/ui/flows.mode.tsx`) are used.

## Exports

Everything is re-exported from `flow.tsx`.

| Group | Export | What it is |
|---|---|---|
| Canvas | `Flow`, `FlowControlButton` | the canvas; xyflow's `ControlButton` for its `controls` slot |
| Node views | `TableNodeView`, `LabelNodeView` | renderers of the `table` node and of a column caption |
| Config elements | `TableNode` | `<TableNode id name ops />`: renders nothing, carries `flowNodeType: 'table'` |
| Node parts | `MetricHint`, `SinkIcon`, `SinkIconTile` | tooltip around a metric; sink-type logo, bare or in a tile |
| Hooks | `useFlowSelection` | flags the selected node/edge, marks animated edges, optional edge `zIndex` |
| | `useSelectionHandlers` | node/edge/pane clicks → selection callbacks |
| | `useEdgeAnimation` | `animatedEdges` plus `animateEdge(id, durationMs?)` |
| | `useMeasuredNodes` | keeps measured node sizes across rebuilt `nodes` |
| Helpers | `collectNodeSpecs`, `columnLabel` | split children into node specs and overlays; a caption node above a column |
| Constants | `TABLE_COLUMN_X`, `SINK_COLUMN_X`, `LABEL_OFFSET_Y`, `DEFAULT_ANIMATE_DURATION` | column geometry; default edge animation time (1300 ms) |

## Usage

A view declares its nodes as config-element children, lays them out itself and passes ready nodes
down (`flow-summary/flow-summary.tsx`):

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

The contract both views follow:

- Config elements are `Object.assign((_props) => null, { flowNodeType })`. `collectNodeSpecs` takes
  them from the direct children in declaration order; the view stacks them into the
  `TABLE_COLUMN_X` / `SINK_COLUMN_X` columns with `TABLES` / `SINKS` captions (`columnLabel`).
  Callers supply no coordinates. Any other child passes through to `<ReactFlow>` as an overlay.
- Selection is controlled by the caller. `useSelectionHandlers` reports clicks on `table` and `sink`
  nodes and on edges only; a click on the selected item or on the empty pane calls
  `onClearSelection`.
- Edge weight is the caller's: `opacity` / `strokeWidth` go into each edge's `style`.
- A view's `ref.animate(...)` runs `useEdgeAnimation`: marching ants for `durationMs` (default
  1300), then revert. A repeat call restarts the timer; every start is phase-locked to a shared
  clock, so a restarted edge does not jump.

Constraints:

- `Flow` is `h-full w-full`: the parent must have a height.
- `Flow` mounts its own `ReactFlowProvider`; children render inside `<ReactFlow>`.
- `nodes`, `edges`, `onNodesChange`, `onMoveStart` are wrapped (transition, measured sizes, user-move
  tracking) and `colorMode` is fixed to `light`; these are set after the props spread. The defaults
  `nodesDraggable={false}`, `nodesConnectable={false}`, `nodesFocusable`, `edgesFocusable`, `fitView`,
  `fitViewOptions={{ maxZoom: 1 }}`, `maxZoom={1}`, `proOptions` come before it and can be overridden.
- Measured sizes are carried over (`useMeasuredNodes`) because React Flow drops a node's handle
  bounds when a new node object arrives without `measured`: edges would unmount for a frame and
  restart their animation on every metrics refresh.
- A layout transition (400 ms) starts only when node ids, node positions or edge ids change; the first
  render is not animated.
- `MetricHint` uses `@/shared/ui/tooltip`, so a `TooltipProvider` must sit above the canvas (mounted
  in `@/app` `Providers`); its child must forward props to a DOM node (`TooltipTrigger asChild`).

## Props

`Flow` takes every `ReactFlow` prop plus its own. Not obvious from the types:

- `revealId` pans that node or edge into view whenever it changes; a change of `revealKey` (defaults
  to `revealId`) re-reveals even if `revealId` stays the same.
- `viewportInsets` are canvas margins covered by floating overlays; reveal and fit view keep them
  clear.
- `controls` render above the zoom buttons, in the same card.

`TableNode` takes `id` plus `TableNodeData`: `name` and `ops`, a `{ count, p95? }` per operation
(`p95` in ms).

## Semantic styles

**Canvas.** `flow.css` maps xyflow's `--xy-*` variables to the design tokens, so light/dark follow
`.dark` (background `--canvas`, edges `--ink-muted`, selected edge `--primary`). Classes callers
rely on:

| Selector | Effect |
|---|---|
| control button `[aria-pressed="true"]` | `--primary` icon: the active button of a toggle group |
| `.flow-controls-joined` | no divider below: buttons of one group sit flush |
| `.selected-target` on a `Handle` | `--primary` border, like the handles of a selected node |

Zoom in / zoom out are disabled at the max / min zoom.

**Table node.** cva `selected`: `border-primary` with a primary glow when selected, `border-hairline`
otherwise. Count tone per operation:

| Operation | Label | Count tone |
|---|---|---|
| `insert` | `INS` | `text-success` |
| `update` | `UPD` | `text-ink` |
| `delete` | `DEL` | `text-danger` |
| `truncate` | `TRN` | `text-ink-muted` |

**Sink icon.** `type` → logo for `kafka`, `nats-core` / `nats-jetstream`, `http`, `rabbitmq` /
`rabbitmq-stream`, plus the ghost types `clickhouse`, `s3`, `snowflake`; any other type falls back to
`Plug`.
