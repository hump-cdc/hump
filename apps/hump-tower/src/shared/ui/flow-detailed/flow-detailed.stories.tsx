import { type ComponentProps, useRef, useState } from 'react'

import type { Meta, StoryObj } from '@storybook/react'
import type { Edge } from '@xyflow/react'
import { fn } from 'storybook/test'

import { Button } from '@/shared/ui/button'

import {
    type ClearSelectionHandler,
    type EdgeSelectHandler,
    FlowDetailed,
    FlowDetailedGhost,
    type FlowDetailedRef,
    FlowDetailedSink,
    FlowDetailedTable,
    type SinkNodeSelectHandler,
    type SinkTargetSelectHandler,
    sinkTargetKey,
    type TableNodeSelectHandler,
} from './flow-detailed'

/**
 * Weighted edge into a sink target (`targetHandle`) or a ghost (no handle):
 * `opacity`/`strokeWidth` encode relative throughput of the route.
 */
const edge = (source: string, target: string, targetHandle: string | undefined, weight: number): Edge => ({
    id: [source, target, targetHandle].filter(Boolean).join('-'),
    source,
    target,
    targetHandle,
    style: { opacity: weight, strokeWidth: 1 + weight * 3 },
})

const edges: Edge[] = [
    edge('t-users', 's-kafka', 'orders', 1),
    edge('t-users', 's-rabbitmq', 'users', 0.7),
    edge('t-users', 'g-clickhouse', undefined, 0.45),
    edge('t-orders', 's-kafka', 'orders', 0.6),
    edge('t-orders', 's-rabbitmq', 'events', 0.85),
    edge('t-orders', 'g-s3', undefined, 0.4),
    edge('t-events', 's-kafka', 'events', 0.4),
    edge('t-events', 's-rabbitmq', 'events', 0.55),
    edge('t-events', 'g-snowflake', undefined, 0.6),
]

type StoryArgs = Pick<ComponentProps<typeof FlowDetailed>, 'edges'> & {
    onTableNodeSelect: TableNodeSelectHandler
    onSinkNodeSelect: SinkNodeSelectHandler
    onSinkTargetSelect: SinkTargetSelectHandler
    onEdgeSelect: EdgeSelectHandler
    onClearSelection: ClearSelectionHandler
}

const meta: Meta<StoryArgs> = {
    title: 'Flows/Flow Detailed',
    parameters: { disableSideBySide: true, disablePadding: true },
    argTypes: {
        edges: { table: { disable: true } },
        onTableNodeSelect: { table: { disable: true } },
        onSinkNodeSelect: { table: { disable: true } },
        onSinkTargetSelect: { table: { disable: true } },
        onEdgeSelect: { table: { disable: true } },
        onClearSelection: { table: { disable: true } },
    },
    args: {
        edges,
        onTableNodeSelect: fn(),
        onSinkNodeSelect: fn(),
        onSinkTargetSelect: fn(),
        onEdgeSelect: fn(),
        onClearSelection: fn(),
    },
    render: ({ onTableNodeSelect, onSinkNodeSelect, onSinkTargetSelect, onEdgeSelect, onClearSelection, ...args }) => {
        const [selected, setSelected] = useState<string | undefined>('s-kafka')
        const flowRef = useRef<FlowDetailedRef>(null)

        const nodeLabel = (id: string) => id.replace(/^[tsg]-/, '')

        return (
            <div className="relative h-full w-full">
                <div className="absolute top-3 left-3 z-10 flex flex-wrap gap-2">
                    {(args.edges ?? []).map((edge) => (
                        <Button
                            key={edge.id}
                            variant="outline"
                            onClick={() =>
                                flowRef.current?.animate(edge.source, edge.target, edge.targetHandle ?? undefined)
                            }
                        >
                            {nodeLabel(edge.source)} → {nodeLabel(edge.target)}
                            {edge.targetHandle ? `:${edge.targetHandle}` : ''}
                        </Button>
                    ))}
                </div>
                <FlowDetailed
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
                    onSinkTargetSelect={(sinkId, target) => {
                        setSelected(sinkTargetKey(sinkId, target.id))
                        onSinkTargetSelect(sinkId, target)
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
                    <FlowDetailedTable
                        id="t-users"
                        name="public.users"
                        ops={{
                            insert: { count: 842, p95: 11 },
                            update: { count: 311, p95: 8 },
                            delete: { count: 47, p95: 19 },
                            truncate: { count: 0, p95: 4 },
                        }}
                    />
                    <FlowDetailedTable
                        id="t-orders"
                        name="public.orders"
                        ops={{
                            insert: { count: 210, p95: 14 },
                            update: { count: 540, p95: 9 },
                            delete: { count: 88, p95: 22 },
                            truncate: { count: 1, p95: 5 },
                        }}
                    />
                    <FlowDetailedTable
                        id="t-events"
                        name="public.events"
                        ops={{
                            insert: { count: 37, p95: 6 },
                            update: { count: 12, p95: 7 },
                            delete: { count: 4, p95: 15 },
                            truncate: { count: 0, p95: 3 },
                        }}
                    />

                    <FlowDetailedSink
                        id="s-kafka"
                        name="Kafka"
                        type="kafka"
                        host="localhost:9092"
                        count={1284}
                        targets={[
                            {
                                id: 'orders',
                                name: 'orders',
                                kind: 'topic',
                                count: 842,
                                p95: 6,
                                successRate: 99.9,
                                health: 'ok',
                            },
                            {
                                id: 'events',
                                name: 'events',
                                kind: 'topic',
                                count: 442,
                                p95: 9,
                                successRate: 99.7,
                                health: 'ok',
                            },
                        ]}
                    />
                    <FlowDetailedSink
                        id="s-rabbitmq"
                        name="RabbitMQ"
                        type="rabbitmq"
                        host="localhost:5672"
                        count={970}
                        targets={[
                            {
                                id: 'users',
                                name: 'users',
                                kind: 'queue',
                                count: 612,
                                p95: 14,
                                successRate: 99.8,
                                health: 'ok',
                            },
                            {
                                id: 'events',
                                name: 'events',
                                kind: 'queue',
                                count: 358,
                                p95: 41,
                                successRate: 97.2,
                                health: 'error',
                            },
                        ]}
                    />

                    <FlowDetailedGhost id="g-clickhouse" type="clickhouse" />
                    <FlowDetailedGhost id="g-s3" type="s3" />
                    <FlowDetailedGhost id="g-snowflake" type="snowflake" />
                </FlowDetailed>
            </div>
        )
    },
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Flow Detailed',
}
