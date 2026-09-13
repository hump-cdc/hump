import { type ReactElement, useCallback } from 'react'

import { GridView } from './data-table.base'
import { useExpandableRows } from './data-table.helpers.expandable'
import type { DataTableClientProps } from './data-table.types'

/** In-memory grid with expandable detail rows. */
export const ClientDataTable = <T,>({
    columns,
    data,
    id,
    getRowId,
    getDetail,
    isRowExpandable,
    detailHeight,
    rowHeight,
    headerHeight,
    className,
    height,
}: DataTableClientProps<T>): ReactElement => {
    const expandable = !!getDetail
    const canExpand = useCallback(
        (row: T) => (isRowExpandable ? isRowExpandable(row) : expandable),
        [isRowExpandable, expandable],
    )

    const { rows, toggle } = useExpandableRows({ data, getRowId, expandable, isRowExpandable: canExpand })

    return (
        <GridView<T>
            columns={columns}
            rows={rows}
            toggle={toggle}
            getDetail={getDetail}
            detailHeight={detailHeight}
            rowHeight={rowHeight}
            headerHeight={headerHeight}
            className={className}
            height={height}
            persistId={id}
        />
    )
}
