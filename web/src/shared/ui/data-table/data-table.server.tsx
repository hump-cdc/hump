import { type ReactElement, useCallback } from 'react'

import { GridView } from './data-table.base'
import { DEFAULT_BLOCK_SIZE } from './data-table.consts'
import { useWindowedRows } from './data-table.helpers.expandable'
import { useInitialBlockLoad, useViewportRangeLoader } from './data-table.helpers.loading'
import type { DataTableServerProps } from './data-table.types'

/**
 * Lazy, server-driven grid on a windowed client-side model. When `rowCount` is
 * known, the grid is seeded with placeholders for a stable scrollbar and the
 * blocks overlapping the viewport are fetched on scroll.
 */
export const ServerDataTable = <T,>({
    columns,
    fetchRows,
    id,
    getRowId,
    rowCount,
    blockSize = DEFAULT_BLOCK_SIZE,
    refreshToken,
    getDetail,
    isRowExpandable,
    detailHeight,
    rowHeight,
    headerHeight,
    className,
    height,
}: DataTableServerProps<T>): ReactElement => {
    const expandable = !!getDetail
    const canExpand = useCallback(
        (row: T) => (isRowExpandable ? isRowExpandable(row) : expandable),
        [isRowExpandable, expandable],
    )

    const { rows, toggle, ensureRange } = useWindowedRows({
        rowCount,
        blockSize,
        fetchRows,
        getRowId,
        expandable,
        isRowExpandable: canExpand,
        refreshToken,
    })

    const handleViewportChanged = useViewportRangeLoader<T>(ensureRange)

    // Kick off the first block so an unknown-total list has something to render.
    useInitialBlockLoad(ensureRange, blockSize)

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
