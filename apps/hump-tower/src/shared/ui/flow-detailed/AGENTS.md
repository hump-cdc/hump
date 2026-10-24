# FlowDetailed

The detailed topology diagram: source tables on the left wired to individual targets (topics,
queues, …) inside each sink on the right, followed by dashed ghost placeholders for sinks that are
not configured. Pick `FlowSummary` (`@/shared/ui/flow-summary`) when one aggregate node per sink is
enough.
Storybook: `Flows/Flow Detailed`.

Exports: `FlowDetailed` (the canvas, built on `Flow` of `@/shared/ui/flow`), the config elements
`FlowDetailedTable` (flow's `TableNode`), `FlowDetailedSink` and `FlowDetailedGhost`,
`sinkTargetKey`, and the data, handler and `FlowDetailedRef` types.

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
    <FlowDetailedTable id="t-users" name="public.users" ops={ops} />
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

Layout, controlled selection, edge `style`, the parent height and `TooltipProvider` follow the
contract in `@/shared/ui/flow` (its `AGENTS.md`, "Usage"). Specific to this view:

- The right column stacks sinks, then ghosts, in declaration order.
- An edge reaches a target through `targetHandle` = the target's `id`; an edge into a ghost has no
  `targetHandle`. Target ids are unique only within their sink, so a target's selection id is
  `sinkTargetKey(sinkId, targetId)` (`<sinkId>:<targetId>`).
- A target row is a button; its click does not reach the sink card, so `onSinkNodeSelect` does not
  fire for it, and a click on the selected target calls `onClearSelection`. A selected target
  reveals (pans to) its whole sink card.
- Ghost nodes are not selectable.
- Edges are drawn above every node (`zIndex` 1001; React Flow lifts a selected node to 1000), so
  they stay visible over the sink card up to the target row's handle.
- Both columns are stacked from height constants in `flow-detailed.consts.ts`, not from measured
  sizes. `TABLE_HEIGHT` and `TABLE_GAP` must match `@/shared/ui/flow`'s `node.table.tsx`;
  `SINK_CHROME_HEIGHT`, `TARGET_ROW_HEIGHT`, `TARGET_GAP`, `GHOST_HEIGHT`, … must match the markup in
  `node.sink.tsx`, `node.sink.target.tsx` and `node.ghost.tsx`. Change the markup, update the
  constants.

## Props

`FlowDetailed` takes the props of `Flow` except `nodes` (so `viewportInsets`, `controls`, any
`ReactFlow` prop) plus its own, typed in `flow-detailed.types.ts`. Not obvious from the types:

- `selected` is a node id, an edge id or a `sinkTargetKey`.
- `ref.animate(tableId, sinkId, targetId?, durationMs?)`: omit `targetId` for an edge into a ghost;
  no-op when there is no such edge.

Config elements:

- `FlowDetailedSink`: `count` is the total across all targets; `host` is shown under the name.
- `SinkTarget`: `kind` is not rendered; `count` is shown only above 0; `p95` (ms) and `successRate`
  (0–100) are hidden when absent; `health` colours the success-rate chip.
- `FlowDetailedGhost`: `type` is shown as the placeholder's icon and label.

## Semantic styles

Canvas, table node and sink icon styles come from `@/shared/ui/flow`.

**Sink card** — cva `selected` and `targetSelected`:

| Variant | Look |
|---|---|
| `selected: true` | `border-primary`, primary glow |
| `selected: false` (default) | `border-hairline`; on hover a neutral glow, and `border-hairline-strong` unless a target row is hovered |
| `targetSelected: true` | primary glow, border stays neutral: one of its target rows is selected |

**Target row** — cva `selected`: `border-primary` and the handle gets `.selected-target` when
selected, `border-hairline` otherwise. The success-rate chip by cva `health`: `ok`
`bg-success-soft text-success`, `error` `bg-danger-soft text-danger`.

**Ghost** — dashed `border-hairline-strong`, `opacity-50`, icon and label `text-ink-subtle`: a sink
that is not configured.
