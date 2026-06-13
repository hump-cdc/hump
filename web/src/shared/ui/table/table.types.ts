import type { ReactNode } from 'react'

export type MobileZone = 'title' | 'meta' | 'fill' | 'badges'

export type TableRowColumn<T> = {
    /** Stable identifier, used as the React key for this column's cells. */
    key: string
    header?: string
    className?: string
    cell: (row: T) => ReactNode
    mobile?: {
        zone: MobileZone
        className?: string
        render: (row: T) => ReactNode
    }
}
