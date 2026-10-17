import type { ReactNode } from 'react'

import { cva } from 'class-variance-authority'

import { Card, CardHeader } from '@/shared/ui/card'
import { cn } from '@/shared/utils'

import type { TableRowColumn } from './table.types'
import { TableContext, type TableVariant } from './table-context'
import { TableRowHeader } from './table-row-header'

const tableHeaderVariants = cva('flex shrink-0 items-center justify-between gap-2', {
    variants: {
        variant: {
            default: 'pb-2',
            bordered: '',
        },
    },
    defaultVariants: {
        variant: 'default',
    },
})

type TableProps<T> = {
    columns: TableRowColumn<T>[]
    /** `default` renders the table without a frame; `bordered` wraps it in a card. */
    variant?: TableVariant
    selectable?: boolean
    header?: ReactNode
    headerAside?: ReactNode
    toolbar?: ReactNode
    children: ReactNode
    className?: string
}

export function Table<T>({
    columns,
    variant = 'default',
    selectable,
    header,
    headerAside,
    toolbar,
    children,
    className,
}: TableProps<T>): ReactNode {
    const hasHeader = Boolean(header || headerAside)
    const body = (
        <>
            {toolbar && (
                <div className={cn('border-hairline border-b', variant === 'bordered' ? 'p-4' : 'py-4')}>{toolbar}</div>
            )}
            <TableRowHeader columns={columns} selectable={selectable} />
            <div>{children}</div>
        </>
    )

    return (
        <TableContext.Provider value={{ variant }}>
            {variant === 'bordered' ? (
                <Card className={cn('overflow-visible', className)}>
                    {hasHeader && <CardHeader aside={headerAside}>{header}</CardHeader>}
                    {body}
                </Card>
            ) : (
                <div className={cn('flex flex-col text-ink', className)}>
                    {hasHeader && (
                        <div className={tableHeaderVariants({ variant })}>
                            {header}
                            {headerAside ? <div className="flex items-center gap-2">{headerAside}</div> : null}
                        </div>
                    )}
                    {body}
                </div>
            )}
        </TableContext.Provider>
    )
}
