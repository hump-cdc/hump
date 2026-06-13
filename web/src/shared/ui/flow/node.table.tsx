import type { FC } from 'react'

import { Handle, type Node, type NodeProps, Position } from '@xyflow/react'
import { cva } from 'class-variance-authority'
import { Table2, TrendingDown, TrendingUp } from 'lucide-react'

import { cn } from '@/shared/utils'

import type { FlowNodeBase, FlowNodeConfig } from './node'

export type TableNodeData = {
    name: string
    rate: string
    trend?: 'up' | 'down'
}

type TableNodeModel = Node<TableNodeData, 'table'>

const tableNodeVariants = cva(
    'flex w-61 flex-col justify-center gap-2.5 rounded-xl border bg-surface-2 px-4 py-3.5 shadow-lg',
    {
        variants: {
            selected: {
                true: 'border-primary',
                false: 'border-hairline',
            },
        },
        defaultVariants: {
            selected: false,
        },
    },
)

export const TableNodeView: FC<NodeProps<TableNodeModel>> = ({ data, selected }) => {
    const Trend = data.trend === 'down' ? TrendingDown : TrendingUp

    return (
        <div className={cn(tableNodeVariants({ selected: Boolean(selected) }))}>
            <div className="flex items-center gap-2.5">
                <Table2 className={cn('size-4', selected ? 'text-primary' : 'text-ink-subtle')} />
                <span className="font-mono text-base font-medium text-ink">{data.name}</span>
            </div>

            <div className="flex items-end">
                <span className="font-mono text-2xl font-semibold text-ink">{data.rate}</span>
                <span className="text-sm font-medium text-ink-subtle pb-0.5">/s</span>
                <div className="flex-1" />
                <Trend className={cn('size-3.5', data.trend === 'down' ? 'text-danger' : 'text-success')} />
            </div>

            <Handle type="source" position={Position.Right} />
        </div>
    )
}

/** Declarative config element: `<TableNode id selected name rate trend />`. */
export const TableNode: FlowNodeConfig<'table', TableNodeData> = Object.assign(
    (_props: FlowNodeBase & TableNodeData) => null,
    { flowNodeType: 'table' as const },
)
