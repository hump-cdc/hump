import type { FC } from 'react'

import { Handle, type Node, type NodeProps, Position } from '@xyflow/react'
import { cva } from 'class-variance-authority'

import { cn } from '@/shared/utils'

import type { FlowNodeBase, FlowNodeConfig } from './node'
import { SinkIconTile } from './sink-icon'

export type SinkNodeData = {
    name: string
    /** Sink type (`kafka`, `rabbitmq`, …); picks the header logo. */
    type: string
    /** Pre-formatted total operations delivered (e.g. `1.24M`). */
    operations: string
    /** Pre-formatted p95 delivery latency (e.g. `18ms`). */
    p95: string
    /** Success rate (0–100). */
    successRate: number
    health: 'ok' | 'error'
}

type SinkNodeModel = Node<SinkNodeData, 'sink'>

const sinkNodeVariants = cva(
    'flex w-61 flex-col gap-3 rounded-xl border bg-surface-1 px-3.5 py-3.5 shadow-lg transition',
    {
        variants: {
            selected: {
                true: 'border-primary shadow-glow shadow-primary/40',
                false: 'border-hairline hover:border-hairline-strong hover:shadow-glow hover:shadow-ink-subtle/30',
            },
        },
        defaultVariants: {
            selected: false,
        },
    },
)

const MetricCell: FC<{ label: string; value: string; className?: string }> = ({ label, value, className }) => (
    <div className="flex flex-col gap-0.5">
        <span className="font-medium text-2xs text-ink-tertiary tracking-caps">{label}</span>
        <span className={cn('font-mono font-semibold text-base text-ink', className)}>{value}</span>
    </div>
)

export const SinkNodeView: FC<NodeProps<SinkNodeModel>> = ({ data, selected }) => (
    <div className={cn(sinkNodeVariants({ selected: Boolean(selected) }))}>
        <div className="flex items-center gap-2.5">
            <SinkIconTile type={data.type} />
            <span className="font-medium font-mono text-ink text-sm">{data.name}</span>
        </div>

        <div className="h-px bg-hairline-strong" />

        <div className="grid grid-cols-3 gap-2">
            <MetricCell label="OPERATIONS" value={data.operations} />
            <MetricCell label="P95" value={data.p95} />
            <MetricCell
                label="SUCCESS"
                value={`${data.successRate}%`}
                className={data.health === 'error' ? 'text-danger' : 'text-success'}
            />
        </div>

        <Handle type="target" position={Position.Left} />
    </div>
)

/** Declarative config element: `<FlowSummarySink id name type operations p95 successRate health />`. */
export const SinkNode: FlowNodeConfig<'sink', SinkNodeData> = Object.assign(
    (_props: FlowNodeBase & SinkNodeData) => null,
    { flowNodeType: 'sink' as const },
)
