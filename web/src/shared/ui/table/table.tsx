import type { ReactNode } from 'react'

import { Card, CardHeader } from '@/shared/ui/card'

import type { TableRowColumn } from './table.types'
import { TableRowHeader } from './table-row-header'

type TableProps<T> = {
    columns: TableRowColumn<T>[]
    selectable?: boolean
    header: ReactNode
    headerAside?: ReactNode
    toolbar?: ReactNode
    children: ReactNode
    className?: string
}

export function Table<T>({
    columns,
    selectable,
    header,
    headerAside,
    toolbar,
    children,
    className,
}: TableProps<T>): ReactNode {
    return (
        <Card className={className}>
            <CardHeader aside={headerAside}>{header}</CardHeader>
            {toolbar && <div className="border-b border-hairline p-4">{toolbar}</div>}
            <TableRowHeader columns={columns} selectable={selectable} />
            <div>{children}</div>
        </Card>
    )
}
