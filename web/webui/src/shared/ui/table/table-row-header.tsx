import type { ReactNode } from 'react'

import { cva } from 'class-variance-authority'

import { cn } from '@/shared/utils'

import type { TableRowColumn } from './table.types'
import { useTableContext } from './table-context'

const tableRowHeaderVariants = cva('hidden items-center border-hairline border-b py-2.5 lg:flex', {
    variants: {
        variant: {
            default: '',
            bordered: 'bg-surface-2 px-4',
        },
        selectable: {
            true: 'gap-3',
            false: 'gap-4',
        },
    },
    defaultVariants: {
        variant: 'default',
        selectable: false,
    },
})

type TableRowHeaderProps<T> = {
    columns: TableRowColumn<T>[]
    selectable?: boolean
}

export function TableRowHeader<T>({ columns, selectable }: TableRowHeaderProps<T>): ReactNode {
    const { variant } = useTableContext()
    const className = tableRowHeaderVariants({ variant, selectable: Boolean(selectable) })
    const headerCell = (col: TableRowColumn<T>) =>
        col.header ? (
            <span
                key={col.key}
                className={cn('font-semibold text-ink-subtle text-xs uppercase tracking-wide', col.className)}
            >
                {col.header}
            </span>
        ) : (
            <span key={col.key} className={col.className} />
        )

    if (selectable) {
        return (
            <div className={className}>
                <span className="size-4.5 shrink-0" />
                <div className="flex flex-1 items-center gap-4">{columns.map(headerCell)}</div>
            </div>
        )
    }

    return <div className={className}>{columns.map(headerCell)}</div>
}
