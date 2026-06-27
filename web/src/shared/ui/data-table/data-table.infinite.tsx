import { type ReactElement, useCallback } from 'react'

import { GridView } from './data-table.base'
import { usePrefixRows } from './data-table.helpers.expandable'
import { useInfiniteScrollLoader } from './data-table.helpers.loading'
import type { DataTableInfiniteProps } from './data-table.types'

export const InfiniteDataTable = <T,>({
    columns,
    data,
    rowCount,
    onLoadMore,
    isFetchingNextPage,
    id,
    getRowId,
    getDetail,
    isRowExpandable,
    detailHeight,
    rowHeight,
    headerHeight,
    className,
    height,
}: DataTableInfiniteProps<T>): ReactElement => {
    const expandable = !!getDetail
    const canExpand = useCallback(
        (row: T) => (isRowExpandable ? isRowExpandable(row) : expandable),
        [isRowExpandable, expandable],
    )

    const { rows, toggle } = usePrefixRows({ data, rowCount, getRowId, expandable, isRowExpandable: canExpand })

    const hasMore = data.length < rowCount && !isFetchingNextPage
    const handleViewportChanged = useInfiniteScrollLoader<T>(onLoadMore, data.length, hasMore)

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
            onViewportChanged={handleViewportChanged}
        />
    )
}
