import type { ReactNode } from 'react'

import { cn } from '@/shared/utils'

import type { TableRowColumn } from './table.types'

type TableRowHeaderProps<T> = {
    columns: TableRowColumn<T>[]
    selectable?: boolean
}

export function TableRowHeader<T>({ columns, selectable }: TableRowHeaderProps<T>): ReactNode {
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
            <div className="hidden items-center gap-3 border-hairline border-b bg-surface-2 px-4 py-2.5 lg:flex">
                <span className="size-4.5 shrink-0" />
                <div className="flex flex-1 items-center gap-4">{columns.map(headerCell)}</div>
            </div>
        )
    }

    return (
        <div className="hidden items-center gap-4 border-hairline border-b bg-surface-2 px-4 py-2.5 lg:flex">
            {columns.map(headerCell)}
        </div>
    )
}
