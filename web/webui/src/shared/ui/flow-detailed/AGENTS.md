# FlowDetailed

The detailed topology diagram: source tables on the left wired to individual targets (topics,
queues, …) inside each sink on the right, followed by dashed ghost placeholders for sinks that are
not configured. Pick `FlowSummary` (`@/shared/ui/flow-summary`) when one aggregate node per sink is
enough.
Storybook: `Flows/Flow Detailed`.

Parts:

- `FlowDetailed` — the canvas, built on `Flow` (`@/shared/ui/flow`).
- `FlowDetailedTable` — config element for a table node (`TableNode` of `@/shared/ui/flow`).
- `FlowDetailedSink` — config element for a sink node with its target rows.
- `FlowDetailedGhost` — config element for a ghost placeholder.
- `sinkTargetKey(sinkId, targetId)` — the selection id of a target row, `<sinkId>:<targetId>`.
- Types: `FlowDetailedRef`, `TableNodeData`, `SinkNodeData`, `SinkTarget`, `GhostNodeData`,
  `TableNodeSelectHandler`, `SinkNodeSelectHandler`, `SinkTargetSelectHandler`, `EdgeSelectHandler`,
  `ClearSelectionHandler`.

## Usage

```tsx
import {
    FlowDetailed,
    FlowDetailedGhost,
    type FlowDetailedRef,
    FlowDetailedSink,
    FlowDetailedTable,
    sinkTargetKey,
} from '@/shared/ui/flow-detailed'

const flowRef = useRef<FlowDetailedRef>(null)
const [selected, setSelected] = useState<string>()

<FlowDetailed
    ref={flowRef}
    edges={[
        { id: 't-users-s-kafka-orders', source: 't-users', target: 's-kafka', targetHandle: 'orders' },
        { id: 't-users-g-clickhouse', source: 't-users', target: 'g-clickhouse' },
    ]}
    selected={selected}
    onTableNodeSelect={(id) => setSelected(id)}
    onSinkNodeSelect={(id) => setSelected(id)}
    onSinkTargetSelect={(sinkId, target) => setSelected(sinkTargetKey(sinkId, target.id))}
    onEdgeSelect={(id) => setSelected(id)}
    onClearSelection={() => setSelected(undefined)}
>
    <FlowDetailedTable
        id="t-users"
        name="public.users"
        ops={{
            insert: { count: 842, p95: 11 },
            update: { count: 311, p95: 8 },
            delete: { count: 47, p95: 19 },
            truncate: { count: 0 },
        }}
    />
    <FlowDetailedSink
        id="s-kafka"
        name="Kafka"
        type="kafka"
        host="localhost:9092"
        count={1284}
        targets={[{ id: 'orders', name: 'orders', kind: 'topic', count: 842, p95: 6, successRate: 99.9, health: 'ok' }]}
    />
    <FlowDetailedGhost id="g-clickhouse" type="clickhouse" />
</FlowDetailed>

flowRef.current?.animate('t-users', 's-kafka', 'orders')
```

- The config elements render nothing; `FlowDetailed` reads them from its direct children and lays
  them out in declaration order: tables in the left column, sinks and then ghosts stacked in the
  right one, columns captioned `TABLES` / `SINKS`. Callers supply no coordinates. Any other child
  passes through to the canvas as an overlay.
- An edge reaches a target through `targetHandle` = the target's `id`; an edge into a ghost has no
  `targetHandle`. Target ids are unique only within their sink.
- Selection is controlled: `selected` is a node id, an edge id, or `sinkTargetKey(sinkId, targetId)`
  for a target row. A click on the selected item or on the empty pane calls `onClearSelection`.
- A target row is a button; its click does not reach the sink card, so `onSinkNodeSelect` does not
  fire for it. A selected target reveals (pans to) its whole sink card.
- Ghost nodes are not selectable.
- Edges are drawn above every node (`zIndex` 1001), so they stay visible over the sink card.
- Edge weight is the caller's: put `opacity` / `strokeWidth` into each edge's `style`.
- The right column is stacked from height constants in `flow-detailed.consts.ts` (`TABLE_HEIGHT`,
  `SINK_CHROME_HEIGHT`, `TARGET_ROW_HEIGHT`, `TARGET_GAP`, `GHOST_HEIGHT`, …), not from measured sizes:
  they must match the markup in `node.sink.tsx`, `node.sink.target.tsx` and `node.ghost.tsx`.
- The parent must have a height, and a `TooltipProvider` must sit above (see `@/shared/ui/flow`).

## Props

`FlowDetailed` — plus the props of `Flow` (`viewportInsets`, `controls`, `className`, any `ReactFlow`
prop) except `nodes`.

| Prop | Type | Default | Description |
|---|---|---|---|
| `edges` | `Edge[]` (xyflow) | — | `source` is a table id, `target` a sink or ghost id, `targetHandle` a target id |
| `selected` | `string` | — | id of the selected node, edge or target (`sinkTargetKey`) |
| `onTableNodeSelect` | `(id: string, data: TableNodeData) => void` | — | a table node was clicked |
| `onSinkNodeSelect` | `(id: string, data: SinkNodeData) => void` | — | a sink card was clicked |
| `onSinkTargetSelect` | `(sinkId: string, target: SinkTarget) => void` | — | a target row was clicked |
| `onEdgeSelect` | `(id: string, edge: { source: string; target: string; targetHandle?: string \| null }) => void` | — | an edge was clicked |
| `onClearSelection` | `() => void` | — | the selected item or the empty pane was clicked |
| `ref` | `Ref<FlowDetailedRef>` | — | imperative handle |

`FlowDetailedRef`

| Method | Description |
|---|---|
| `animate(tableId, sinkId, targetId?, durationMs?)` | dashes the edge from `tableId` to `targetId` of `sinkId` as marching ants for `durationMs` (default 1300), then reverts; omit `targetId` for an edge into a ghost; no-op if there is no such edge |

`FlowDetailedTable` — `id`, `name`, `ops`; see `TableNode` in `@/shared/ui/flow`.

`FlowDetailedSink`

| Prop | Type | Default | Description |
|---|---|---|---|
| `id` | `string` | required | node id, referenced by edges and `selected` |
| `name` | `string` | required | sink name |
| `type` | `string` | required | sink type (`kafka`, `rabbitmq`, …); picks the header logo |
| `host` | `string` | — | connection host, shown under the name |
| `count` | `number` | required | total operations across all targets |
| `targets` | `SinkTarget[]` | required | one row per target |

`SinkTarget`

| Field | Type | Default | Description |
|---|---|---|---|
| `id` | `string` | required | handle id, referenced by an edge's `targetHandle` |
| `name` | `string` | required | row label |
| `kind` | `string` | required | target kind (`topic`, `queue`); not rendered by the row |
| `count` | `number` | required | operations delivered; shown only when above 0 |
| `p95` | `number` | — | p95 delivery latency, ms; hidden when absent |
| `successRate` | `number` | — | success rate, 0–100; hidden when absent |
| `health` | `'ok' \| 'error'` | required | colours the success-rate chip |

`FlowDetailedGhost`

| Prop | Type | Default | Description |
|---|---|---|---|
| `id` | `string` | required | node id, referenced by edges |
| `type` | `string` | required | sink type shown as the placeholder: icon and label |

## Semantic styles

Canvas, table node and sink icon styles come from `@/shared/ui/flow`.

**Sink card** — cva `selected` and `targetSelected`, on a `bg-surface-1` card:

| Variant | Classes | Meaning |
|---|---|---|
| `selected: true` | `border-primary shadow-glow shadow-primary/40` | the card itself is selected |
| `selected: false` (default) | `border-hairline`; hover `shadow-glow shadow-ink-subtle/30`, and `border-hairline-strong` unless a target row is hovered | idle |
| `targetSelected: true` | `shadow-glow shadow-primary/40`, border stays neutral | one of its target rows is selected |

**Target row** — cva `selected`, on `bg-surface-3/30`:

| `selected` | Classes |
|---|---|
| `true` | `border-primary`; its handle gets `.selected-target` (primary border) |
| `false` (default) | `border-hairline`, hover `border-hairline-strong` |

Row chips: count `text-ink-subtle`; p95 `bg-surface-2 border-hairline-strong text-ink-subtle`;
success rate by cva `health`:

| `health` | Classes |
|---|---|
| `ok` | `bg-success-soft text-success` |
| `error` | `bg-danger-soft text-danger` |

**Ghost** — `border-hairline-strong border-dashed bg-surface-1 opacity-50`, icon and label
`text-ink-subtle`: a sink that is not configured.
