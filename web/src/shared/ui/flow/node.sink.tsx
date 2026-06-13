import type { FC } from 'react'

import { Handle, type Node, type NodeProps, Position } from '@xyflow/react'
import { cva } from 'class-variance-authority'

import { cn } from '@/shared/utils'

import type { FlowNodeBase, FlowNodeConfig } from './node'

export type SinkNodeData = {
    name: string
    subtitle?: string
    rate: string
    health: 'ok' | 'error'
    /** Success rate (0–100). Renders as the green section; failure (red) fills the remaining `100 - successRate`. */
    successRate: number
}

type SinkNodeModel = Node<SinkNodeData, 'sink'>

const sinkNodeVariants = cva('flex w-75 flex-col overflow-hidden rounded-xl border bg-surface-2 shadow-lg', {
    variants: {
        selected: {
            true: 'border-primary',
            false: 'border-hairline',
        },
    },
    defaultVariants: {
        selected: false,
    },
})

export const SinkNodeView: FC<NodeProps<SinkNodeModel>> = ({ data, selected }) => {
    const isError = data.health === 'error'

    return (
        <div className={cn(sinkNodeVariants({ selected: Boolean(selected) }))}>
            <div className="flex flex-1 items-center justify-between px-4 py-3">
                <div className="flex flex-col gap-1">
                    <span className="font-mono text-md font-medium text-ink">{data.name}</span>
                    {data.subtitle ? <span className="text-xs text-ink-subtle">{data.subtitle}</span> : null}
                </div>

                <div className="flex flex-col items-start gap-1.5">
                    <span className={cn('font-mono text-base font-medium', isError ? 'text-danger' : 'text-ink')}>
                        {data.rate}
                    </span>
                </div>
            </div>

            <div className="flex h-1.5 w-full bg-danger">
                <div
                    className="h-full bg-success"
                    style={{ width: `${Math.max(0, Math.min(100, data.successRate))}%` }}
                />
            </div>

            <Handle type="target" position={Position.Left} />
        </div>
    )
}

/** Declarative config element: `<SinkNode id selected name subtitle rate health successRate />`. */
export const SinkNode: FlowNodeConfig<'sink', SinkNodeData> = Object.assign(
    (_props: FlowNodeBase & SinkNodeData) => null,
    { flowNodeType: 'sink' as const },
)
