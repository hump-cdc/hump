# FlowSummary

The summary topology diagram: source tables on the left wired to one aggregate node per sink on the
right. Pick it for the overview; pick `FlowDetailed` (`@/shared/ui/flow-detailed`) when edges must
land on individual targets (topics, queues) inside a sink.
Storybook: `Flows/Flow Summary`.

Exports: `FlowSummary` (the canvas, built on `Flow` of `@/shared/ui/flow`), the config elements
`FlowSummaryTable` (flow's `TableNode`) and `FlowSummarySink`, and the data, handler and
`FlowSummaryRef` types.

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
    <FlowSummaryTable id="t-users" name="public.users" ops={ops} />
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

Layout, controlled selection, edge `style`, the parent height and `TooltipProvider` follow the
contract in `@/shared/ui/flow` (its `AGENTS.md`, "Usage"). `selected` is a node or edge id; the
selected item is panned into view (`revealId` is set to `selected`).

## Props

`FlowSummary` takes the props of `Flow` except `nodes` (so `viewportInsets`, `controls`, any
`ReactFlow` prop) plus its own, typed in `flow-summary.types.ts`.
`ref.animate(tableId, sinkId, durationMs?)` is a no-op when there is no such edge.

`FlowSummarySink`: `operations` is the total delivered; `p95` (ms) and `successRate` (0–100) render a
dash when absent; `health` colours the success rate.

## Semantic styles

Canvas, table node and sink icon styles come from `@/shared/ui/flow`.

**Sink node** — cva `selected`: `border-primary` with a primary glow when selected,
`border-hairline` otherwise. Metric cells `OPERATIONS`, `P95`, `SUCCESS`; the `SUCCESS` value is
coloured by data:

| Condition | Token |
|---|---|
| `successRate` is `undefined` | `text-ink-subtle` |
| `health === 'error'` | `text-danger` |
| otherwise | `text-success` |
