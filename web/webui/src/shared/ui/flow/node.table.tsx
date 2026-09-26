import type { FC } from 'react'

import { Handle, type Node, type NodeProps, Position } from '@xyflow/react'
import { cva } from 'class-variance-authority'
import { Table2 } from 'lucide-react'

import { NumberFlow } from '@/shared/ui/number-flow'
import { cn } from '@/shared/utils'

import type { FlowNodeBase, FlowNodeConfig } from './node'

/** A single operation-kind cell: count plus optional p95 latency in milliseconds. */
type TableOpStat = {
    count: number
    p95?: number
}

export type TableNodeData = {
    name: string
    ops: Record<'insert' | 'update' | 'delete' | 'truncate', TableOpStat>
}

type TableNodeModel = Node<TableNodeData, 'table'>

const tableNodeVariants = cva(
    'flex w-70 flex-col gap-2.5 rounded-xl border bg-surface-2 px-4 py-2.5 shadow-lg transition',
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

/** Operation cells in display order, each with its label and value tone. */
const OP_CELLS = [
    { key: 'insert', label: 'INS', tone: 'text-success' },
    { key: 'update', label: 'UPD', tone: 'text-ink' },
    { key: 'delete', label: 'DEL', tone: 'text-danger' },
    { key: 'truncate', label: 'TRN', tone: 'text-ink-muted' },
] as const

export const TableNodeView: FC<NodeProps<TableNodeModel>> = ({ data, selected }) => (
    <div className={cn(tableNodeVariants({ selected: Boolean(selected) }))}>
        <div className="flex items-center gap-2.5">
            <Table2 className={cn('size-4', selected ? 'text-primary' : 'text-ink-subtle')} />
            <span className="font-medium font-mono text-ink text-sm leading-tight">{data.name}</span>
        </div>

        <div className="h-px bg-hairline-strong" />

        <div className="grid grid-cols-4 gap-x-4 gap-y-2">
            {OP_CELLS.map(({ key, label, tone }) => (
                <div key={key} className="flex flex-col gap-0.5">
                    <span className="font-medium text-2xs text-ink-tertiary leading-none tracking-caps">{label}</span>
                    <span className={cn('font-mono font-semibold text-base leading-tight', tone)}>
                        <NumberFlow value={data.ops[key].count} kind="compact" />
                    </span>
                    <span className="font-mono text-2xs text-ink-subtle leading-none">
                        <NumberFlow value={data.ops[key].p95} kind="latency" />
                    </span>
                </div>
            ))}
        </div>

        <Handle type="source" position={Position.Right} />
    </div>
)

/** Declarative config element: `<TableNode id name ops />`. */
export const TableNode: FlowNodeConfig<'table', TableNodeData> = Object.assign(
    (_props: FlowNodeBase & TableNodeData) => null,
    { flowNodeType: 'table' as const },
)
