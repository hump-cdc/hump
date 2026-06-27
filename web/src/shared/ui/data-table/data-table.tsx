import type { ReactElement } from 'react'

import { ClientDataTable } from './data-table.client'
import { InfiniteDataTable } from './data-table.infinite'
import { ServerDataTable } from './data-table.server'
import type { DataTableInfiniteProps, DataTableProps, DataTableServerProps } from './data-table.types'

/**
 * A virtualized, resizable data table built on AG Grid Community.
 *
 * - Client mode (`data`) renders in-memory rows.
 * - Server mode (`fetchRows`) lazily loads blocks on scroll into a windowed
 *   client-side model; pass `rowCount` for a stable, up-front scrollbar.
 * - Infinite mode (`onLoadMore`) renders an externally paginated prefix (`data`
 *   + `rowCount`), padding the tail with placeholders and requesting more on scroll.
 */
const isInfiniteMode = <T,>(props: DataTableProps<T>): props is DataTableInfiniteProps<T> =>
    typeof (props as DataTableInfiniteProps<T>).onLoadMore === 'function'

const isServerMode = <T,>(props: DataTableProps<T>): props is DataTableServerProps<T> =>
    typeof (props as DataTableServerProps<T>).fetchRows === 'function'

export const DataTable = <T,>(props: DataTableProps<T>): ReactElement => {
    if (isInfiniteMode(props)) {
        return <InfiniteDataTable<T> {...props} />
    }
    return isServerMode(props) ? <ServerDataTable<T> {...props} /> : <ClientDataTable<T> {...props} />
}
