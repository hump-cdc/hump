import type { FC } from 'react'

import { Handle, type Node, type NodeProps, Position } from '@xyflow/react'
import { cva } from 'class-variance-authority'

import { cn } from '@/shared/utils'

import { sinkTargetKey, useTargetSelection } from './flow-detailed.context'
import type { FlowNodeBase, FlowNodeConfig } from './node'
import { SinkIconTile } from './sink-icon'

/** A delivery target inside a sink (topic, queue, subject, …); edges attach to it via `targetHandle`. */
export type SinkTarget = {
    /** Handle id; referenced by an edge's `targetHandle`. Unique within the sink. */
    id: string
    name: string
    /** Target kind label (e.g. `topic`, `queue`). */
    kind: string
    /** Pre-formatted operations delivered to this target. */
    count: string
    /** Pre-formatted p95 delivery latency (e.g. `6ms`). */
    p95: string
    /** Success rate (0–100). */
    successRate: number
    health: 'ok' | 'error'
}

export type SinkNodeData = {
    name: string
    /** Sink type (`kafka`, `rabbitmq`, …); picks the header logo. */
    type: string
    host?: string
    /** Pre-formatted total operations across all targets. */
    count: string
    targets: SinkTarget[]
}

type SinkNodeModel = Node<SinkNodeData, 'sink'>

const sinkNodeVariants = cva('flex w-95 flex-col gap-2.5 rounded-xl border bg-surface-1 p-3 shadow-lg transition', {
    variants: {
        selected: {
            true: 'border-primary shadow-glow shadow-primary/40',
            // The glow shows on any hover (target rows included); the border reacts to the card only.
            false: [
                'border-hairline hover:not-has-[button:hover]:border-hairline-strong',
                'hover:shadow-glow hover:shadow-ink-subtle/30',
            ],
        },
        /** A target row inside is selected: the card glows in the accent color, its border stays neutral. */
        targetSelected: {
            true: 'shadow-glow shadow-primary/40 hover:shadow-primary/40',
            false: '',
        },
    },
    defaultVariants: {
        selected: false,
        targetSelected: false,
    },
})

const rateVariants = cva('rounded px-1.5 py-0.5 font-medium font-mono text-2xs', {
    variants: {
        health: {
            ok: 'bg-success-soft text-success',
            error: 'bg-danger-soft text-danger',
        },
    },
})

const targetRowVariants = cva(
    'flex h-7 w-full cursor-pointer items-center gap-2 rounded-md border bg-surface-3 px-2.5 text-left',
    {
        variants: {
            selected: {
                true: 'border-primary',
                false: 'border-hairline hover:border-hairline-strong',
            },
        },
        defaultVariants: {
            selected: false,
        },
    },
)

/**
 * One target row; owns the target handle so edges land on the row itself. The row is a
 * button: activating it selects the target instead of the sink card (the click doesn't
 * bubble to the node). The handle sits beside the button since a button can't contain it.
 */
const TargetRow: FC<{ sinkId: string; target: SinkTarget }> = ({ sinkId, target }) => {
    const { selected, onTargetClick } = useTargetSelection()
    const isSelected = selected === sinkTargetKey(sinkId, target.id)

    return (
        <div className="relative">
            <button
                type="button"
                className={targetRowVariants({ selected: isSelected })}
                onClick={(event) => {
                    event.stopPropagation()
                    onTargetClick(sinkId, target)
                }}
            >
                <span className="font-medium font-mono text-ink text-sm">{target.name}</span>
                <span className="text-2xs text-ink-subtle">{target.kind}</span>
                <span className="flex-1" />
                <span className="font-mono text-ink-subtle text-xs">{target.count}</span>
                <span className="rounded border border-hairline-strong bg-surface-2 px-1.5 font-mono text-ink-subtle text-xs">
                    {target.p95}
                </span>
                <span className={rateVariants({ health: target.health })}>{target.successRate}%</span>
            </button>

            <Handle
                type="target"
                id={target.id}
                position={Position.Left}
                className={isSelected ? 'selected-target' : undefined}
            />
        </div>
    )
}

export const SinkNodeView: FC<NodeProps<SinkNodeModel>> = ({ id, data, selected }) => {
    const { selected: selectedId } = useTargetSelection()
    const targetSelected = data.targets.some((target) => sinkTargetKey(id, target.id) === selectedId)

    return (
        <div className={cn(sinkNodeVariants({ selected: Boolean(selected), targetSelected }))}>
            <div className="flex h-8.5 items-center gap-2.5">
                <SinkIconTile type={data.type} />
                <div className="flex flex-col gap-0.5">
                    <span className="font-medium text-ink text-sm leading-tight">{data.name}</span>
                    {data.host ? (
                        <span className="font-mono text-ink-subtle text-xs leading-tight">{data.host}</span>
                    ) : null}
                </div>
                <div className="flex-1" />
                <span className="font-mono font-semibold text-base text-ink">{data.count}</span>
            </div>

            <div className="h-px bg-hairline-strong" />

            <div className="flex flex-col gap-2">
                {data.targets.map((target) => (
                    <TargetRow key={target.id} sinkId={id} target={target} />
                ))}
            </div>
        </div>
    )
}

/** Declarative config element: `<FlowDetailedSink id name type host count targets />`. */
export const SinkNode: FlowNodeConfig<'sink', SinkNodeData> = Object.assign(
    (_props: FlowNodeBase & SinkNodeData) => null,
    { flowNodeType: 'sink' as const },
)
