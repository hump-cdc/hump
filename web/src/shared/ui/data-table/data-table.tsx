import type { ReactElement } from 'react'

import { ClientDataTable } from './data-table.client'
import { ServerDataTable } from './data-table.server'
import type { DataTableProps, DataTableServerProps } from './data-table.types'

/**
 * A virtualized, resizable data table built on AG Grid Community.
 *
 * - Client mode (`data`) renders in-memory rows.
 * - Server mode (`fetchRows`) lazily loads blocks on scroll into a windowed
 *   client-side model; pass `rowCount` for a stable, up-front scrollbar.
 */
const isServerMode = <T,>(props: DataTableProps<T>): props is DataTableServerProps<T> =>
    typeof (props as DataTableServerProps<T>).fetchRows === 'function'

export const DataTable = <T,>(props: DataTableProps<T>): ReactElement =>
    isServerMode(props) ? <ServerDataTable<T> {...props} /> : <ClientDataTable<T> {...props} />
