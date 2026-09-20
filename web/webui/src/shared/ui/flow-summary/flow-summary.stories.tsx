import { type ComponentProps, useRef, useState } from 'react'

import type { Meta, StoryObj } from '@storybook/react'
import type { Edge } from '@xyflow/react'
import { fn } from 'storybook/test'

import { Button } from '@/shared/ui/button'

import {
    type ClearSelectionHandler,
    type EdgeSelectHandler,
    FlowSummary,
    type FlowSummaryRef,
    FlowSummarySink,
    FlowSummaryTable,
    type SinkNodeSelectHandler,
    type TableNodeSelectHandler,
} from './flow-summary'

/** Weighted edge: `opacity`/`strokeWidth` encode relative throughput of the route. */
const edge = (source: string, target: string, weight: number): Edge => ({
    id: `${source}-${target}`,
    source,
    target,
    style: { opacity: weight, strokeWidth: 1 + weight * 3 },
})

const edges: Edge[] = [
    edge('t-users', 's-kafka', 0.85),
    edge('t-users', 's-rabbitmq', 0.5),
    edge('t-orders', 's-kafka', 0.7),
    edge('t-orders', 's-rabbitmq', 0.85),
    edge('t-events', 's-kafka', 0.5),
    edge('t-events', 's-rabbitmq', 0.7),
]

type StoryArgs = Pick<ComponentProps<typeof FlowSummary>, 'edges'> & {
    onTableNodeSelect: TableNodeSelectHandler
    onSinkNodeSelect: SinkNodeSelectHandler
    onEdgeSelect: EdgeSelectHandler
    onClearSelection: ClearSelectionHandler
}

const meta: Meta<StoryArgs> = {
    title: 'UI Kit/Flow Summary',
    parameters: { disableSideBySide: true, disablePadding: true },
    argTypes: {
        edges: { table: { disable: true } },
        onTableNodeSelect: { table: { disable: true } },
        onSinkNodeSelect: { table: { disable: true } },
        onEdgeSelect: { table: { disable: true } },
        onClearSelection: { table: { disable: true } },
    },
    args: {
        edges,
        onTableNodeSelect: fn(),
        onSinkNodeSelect: fn(),
        onEdgeSelect: fn(),
        onClearSelection: fn(),
    },
    render: ({ onTableNodeSelect, onSinkNodeSelect, onEdgeSelect, onClearSelection, ...args }) => {
        const [selected, setSelected] = useState<string | undefined>('s-kafka')
        const flowRef = useRef<FlowSummaryRef>(null)

        const nodeLabel = (id: string) => id.replace(/^[ts]-/, '')

        return (
            <div className="relative h-full w-full">
                <div className="absolute top-3 left-3 z-10 flex flex-wrap gap-2">
                    {(args.edges ?? []).map((edge) => (
                        <Button
                            key={edge.id}
                            variant="outline"
                            onClick={() => flowRef.current?.animate(edge.source, edge.target)}
                        >
                            {nodeLabel(edge.source)} → {nodeLabel(edge.target)}
                        </Button>
                    ))}
                </div>
                <FlowSummary
                    {...args}
                    ref={flowRef}
                    selected={selected}
                    onTableNodeSelect={(id, data) => {
                        setSelected(id)
                        onTableNodeSelect(id, data)
                    }}
                    onSinkNodeSelect={(id, data) => {
                        setSelected(id)
                        onSinkNodeSelect(id, data)
                    }}
                    onEdgeSelect={(id, edge) => {
                        setSelected(id)
                        onEdgeSelect(id, edge)
                    }}
                    onClearSelection={() => {
                        setSelected(undefined)
                        onClearSelection()
                    }}
                >
                    <FlowSummaryTable
                        id="t-users"
                        name="public.users"
                        ops={{
                            insert: { count: '842', p95: '11ms' },
                            update: { count: '311', p95: '8ms' },
                            delete: { count: '47', p95: '19ms' },
                            truncate: { count: '0', p95: '4ms' },
                        }}
                    />
                    <FlowSummaryTable
                        id="t-orders"
                        name="public.orders"
                        ops={{
                            insert: { count: '210', p95: '14ms' },
                            update: { count: '540', p95: '9ms' },
                            delete: { count: '88', p95: '22ms' },
                            truncate: { count: '1', p95: '5ms' },
                        }}
                    />
                    <FlowSummaryTable
                        id="t-events"
                        name="public.events"
                        ops={{
                            insert: { count: '37', p95: '6ms' },
                            update: { count: '12', p95: '7ms' },
                            delete: { count: '4', p95: '15ms' },
                            truncate: { count: '0', p95: '3ms' },
                        }}
                    />

                    <FlowSummarySink
                        id="s-kafka"
                        name="kafka"
                        type="kafka"
                        operations="1.24M"
                        p95="18ms"
                        successRate={99.98}
                        health="ok"
                    />
                    <FlowSummarySink
                        id="s-rabbitmq"
                        name="rabbitmq"
                        type="rabbitmq"
                        operations="486K"
                        p95="31ms"
                        successRate={99.7}
                        health="ok"
                    />
                </FlowSummary>
            </div>
        )
    },
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Flow Summary',
}
