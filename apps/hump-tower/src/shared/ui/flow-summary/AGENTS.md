# FlowSummary

The summary topology diagram: source tables on the left wired to one aggregate node per sink on the
right. Pick it for the overview; pick `FlowDetailed` (`@/shared/ui/flow-detailed`) when edges must
land on individual targets (topics, queues) inside a sink.
Storybook: `Flows/Flow Summary`.

Parts:

- `FlowSummary` — the canvas, built on `Flow` (`@/shared/ui/flow`).
- `FlowSummaryTable` — config element for a table node (`TableNode` of `@/shared/ui/flow`).
- `FlowSummarySink` — config element for a sink node.
- Types: `FlowSummaryRef`, `TableNodeData`, `SinkNodeData`, `TableNodeSelectHandler`,
  `SinkNodeSelectHandler`, `EdgeSelectHandler`, `ClearSelectionHandler`.

## Usage

```tsx
import { FlowSummary, type FlowSummaryRef, FlowSummarySink, FlowSummaryTable } from '@/shared/ui/flow-summary'

const flowRef = useRef<FlowSummaryRef>(null)
const [selected, setSelected] = useState<string>()

<FlowSummary
    ref={flowRef}
    edges={[{ id: 't-users-s-kafka', source: 't-users', target: 's-kafka', style: { opacity: 0.85, strokeWidth: 3.5 } }]}
    selected={selected}
    onTableNodeSelect={(id) => setSelected(id)}
    onSinkNodeSelect={(id) => setSelected(id)}
    onEdgeSelect={(id) => setSelected(id)}
    onClearSelection={() => setSelected(undefined)}
>
    <FlowSummaryTable
        id="t-users"
        name="public.users"
        ops={{
            insert: { count: 842, p95: 11 },
            update: { count: 311, p95: 8 },
            delete: { count: 47, p95: 19 },
            truncate: { count: 0 },
        }}
    />
    <FlowSummarySink
        id="s-kafka"
        name="kafka"
        type="kafka"
        host="localhost:9092"
        operations={1240000}
        p95={18}
        successRate={99.98}
        health="ok"
    />
</FlowSummary>

flowRef.current?.animate('t-users', 's-kafka')
```

- `FlowSummaryTable` and `FlowSummarySink` render nothing; `FlowSummary` reads them from its direct
  children and lays them out in declaration order: tables in the left column, sinks in the right,
  each column captioned `TABLES` / `SINKS`. Callers supply no coordinates. Any other child passes
  through to the canvas as an overlay.
- Selection is controlled: `selected` is the id of a node or an edge; the component only reports
  clicks. A click on the selected item or on the empty pane calls `onClearSelection`.
- The selected item is panned into view (`revealId` is set to `selected`).
- Edge weight is the caller's: put `opacity` / `strokeWidth` into each edge's `style`.
- The parent must have a height, and a `TooltipProvider` must sit above (see `@/shared/ui/flow`).

## Props

`FlowSummary` — plus the props of `Flow` (`viewportInsets`, `controls`, `className`, any `ReactFlow`
prop) except `nodes`.

| Prop | Type | Default | Description |
|---|---|---|---|
| `edges` | `Edge[]` (xyflow) | — | `source` is a table id, `target` a sink id |
| `selected` | `string` | — | id of the selected node or edge |
| `onTableNodeSelect` | `(id: string, data: TableNodeData) => void` | — | a table node was clicked |
| `onSinkNodeSelect` | `(id: string, data: SinkNodeData) => void` | — | a sink node was clicked |
| `onEdgeSelect` | `(id: string, edge: { source: string; target: string }) => void` | — | an edge was clicked |
| `onClearSelection` | `() => void` | — | the selected item or the empty pane was clicked |
| `ref` | `Ref<FlowSummaryRef>` | — | imperative handle |

`FlowSummaryRef`

| Method | Description |
|---|---|
| `animate(tableId, sinkId, durationMs?)` | dashes the edge from `tableId` to `sinkId` as marching ants for `durationMs` (default 1300), then reverts; no-op if there is no such edge |

`FlowSummaryTable` — `id`, `name`, `ops`; see `TableNode` in `@/shared/ui/flow`.

`FlowSummarySink`

| Prop | Type | Default | Description |
|---|---|---|---|
| `id` | `string` | required | node id, referenced by edges and `selected` |
| `name` | `string` | required | sink name |
| `type` | `string` | required | sink type (`kafka`, `rabbitmq`, …); picks the header logo |
| `host` | `string` | — | connection host, shown under the name |
| `operations` | `number` | required | total operations delivered |
| `p95` | `number` | — | p95 delivery latency, ms |
| `successRate` | `number` | — | success rate, 0–100 |
| `health` | `'ok' \| 'error'` | required | colours the success rate |

## Semantic styles

Canvas, table node and sink icon styles come from `@/shared/ui/flow`.

**Sink node** — cva `selected`, on a `bg-surface-1` card:

| `selected` | Classes |
|---|---|
| `true` | `border-primary shadow-glow shadow-primary/40` |
| `false` (default) | `border-hairline`, hover `border-hairline-strong shadow-glow shadow-ink-subtle/30` |

Metric cells (`OPERATIONS`, `P95`, `SUCCESS`): label `text-ink-tertiary`, value `text-ink`. The
`SUCCESS` value is coloured by data:

| Condition | Token |
|---|---|
| `successRate` is `undefined` | `text-ink-subtle` |
| `health === 'error'` | `text-danger` |
| otherwise | `text-success` |
