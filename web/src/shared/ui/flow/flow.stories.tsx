import { type ComponentProps, useRef, useState } from 'react'

import type { Meta, StoryObj } from '@storybook/react'
import type { Edge } from '@xyflow/react'
import { fn } from 'storybook/test'

import { Button } from '@/shared/ui/button'

import {
    type ClearSelectionHandler,
    type EdgeSelectHandler,
    Flow,
    type FlowRef,
    type SinkNodeSelectHandler,
    type TableNodeSelectHandler,
} from './flow'
import { SinkNode } from './node.sink'
import { TableNode } from './node.table'

const edge = (source: string, target: string, opacity: number, strokeWidth: number): Edge => ({
    id: `${source}-${target}`,
    source,
    target,
    style: { opacity, strokeWidth },
})

const edges: Edge[] = [
    edge('t-users', 's-kafka', 1, 5),
    edge('t-users', 's-natsc', 0.9, 3),
    edge('t-users', 's-natsj', 0.7, 2.5),
    edge('t-users', 's-rabbit', 0.5, 2.5),
    edge('t-users', 's-webhook', 0.55, 2),
    edge('t-posts', 's-kafka', 0.45, 2),
    edge('t-posts', 's-natsj', 0.6, 2),
    edge('t-events', 's-rabbit', 0.3, 2),
    edge('t-events', 's-webhook', 0.25, 1.5),
]

type StoryArgs = Pick<ComponentProps<typeof Flow>, 'edges'> & {
    onTableNodeSelect: TableNodeSelectHandler
    onSinkNodeSelect: SinkNodeSelectHandler
    onEdgeSelect: EdgeSelectHandler
    onClearSelection: ClearSelectionHandler
}

const meta: Meta<StoryArgs> = {
    title: 'UI Kit/Flow',
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
        const flowRef = useRef<FlowRef>(null)

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
                <Flow
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
                    <TableNode id="t-users" name="public.users" rate="842" trend="up" />
                    <TableNode id="t-posts" name="public.posts" rate="210" trend="up" />
                    <TableNode id="t-events" name="public.events" rate="37" trend="up" />

                    <SinkNode id="s-kafka" name="kafka1" subtitle="kafka" rate="842/s" health="ok" successRate={100} />
                    <SinkNode id="s-natsc" name="nats-core" subtitle="nats" rate="840/s" health="ok" successRate={99} />
                    <SinkNode
                        id="s-natsj"
                        name="nats-jetstream"
                        subtitle="jetstream"
                        rate="839/s"
                        health="ok"
                        successRate={80}
                    />
                    <SinkNode
                        id="s-rabbit"
                        name="rabbitmq1"
                        subtitle="amqp"
                        rate="211/s"
                        health="error"
                        successRate={25}
                    />
                    <SinkNode
                        id="s-webhook"
                        name="webhook-http"
                        subtitle="http"
                        rate="588/s"
                        health="ok"
                        successRate={99}
                    />
                </Flow>
            </div>
        )
    },
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Flow',
}
