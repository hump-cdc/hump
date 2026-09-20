import { memo, type ReactNode } from 'react'

import { cn } from '@/shared/utils'

import { EXPAND_COLUMN_WIDTH } from './data-table.consts'
import { ExpandButton } from './data-table.expand-button'
import { Skeleton } from './data-table.skeleton'
import type { DataTableColumn, DetailComponent } from './data-table.types'

type DataTableRowProps<T> = {
    columns: DataTableColumn<T>[]
    widths: number[]
    totalWidth: number
    /** Width of the visible body, which the detail panel spans regardless of horizontal scroll. */
    viewportWidth: number
    /** Loaded row, or `undefined` to draw skeleton cells. */
    row: T | undefined
    rowId: string | undefined
    start: number
    rowHeight: number
    isExpandable: boolean
    canExpand: boolean
    isExpanded: boolean
    detailHeight: number
    Detail: DetailComponent<T> | undefined
    onToggle: (id: string) => void
    isSelected: boolean
    /** Called with the row's id and the row when it is clicked; rows are not selectable without it. */
    onSelect: ((id: string, row: T) => void) | undefined
}

/**
 * One absolutely positioned body row and, when expanded, the detail panel under
 * it. Memoized, so scrolling only re-renders rows whose position or data changed.
 */
export const DataTableRow = memo(
    <T,>({
        columns,
        widths,
        totalWidth,
        viewportWidth,
        row,
        rowId,
        start,
        rowHeight,
        isExpandable,
        canExpand,
        isExpanded,
        detailHeight,
        Detail,
        onToggle,
        isSelected,
        onSelect,
    }: DataTableRowProps<T>): ReactNode => (
        <div
            data-slot="data-table-row"
            className="absolute top-0 left-0"
            style={{ transform: `translateY(${start}px)`, width: totalWidth }}
        >
            {/* biome-ignore lint/a11y/noStaticElementInteractions: rows are keyboard-driven from the table body (arrows, Space), not one by one. */}
            {/* biome-ignore lint/a11y/useKeyWithClickEvents: same as above. */}
            <div
                onClick={onSelect && row !== undefined && rowId !== undefined ? () => onSelect(rowId, row) : undefined}
                className={cn(
                    'flex border-hairline border-b bg-surface-1 hover:bg-surface-2',
                    onSelect && rowId !== undefined && 'cursor-pointer',
                    isSelected && 'bg-primary-soft/40 hover:bg-primary-soft/40',
                )}
                style={{ height: rowHeight }}
            >
                {isExpandable && (
                    <div className={cellClass} style={{ width: EXPAND_COLUMN_WIDTH, padding: 0 }}>
                        {row !== undefined && rowId !== undefined && canExpand && (
                            <ExpandButton isExpanded={isExpanded} onToggle={() => onToggle(rowId)} />
                        )}
                    </div>
                )}
                {columns.map((column, index) => (
                    <div
                        // biome-ignore lint/suspicious/noArrayIndexKey: columns are positional, like their ids.
                        key={index}
                        className={cn(cellClass, column.cellClass)}
                        style={{ width: widths[index], lineHeight: `${rowHeight - 5}px` }}
                    >
                        {row === undefined ? <Skeleton /> : renderCell(column, row)}
                    </div>
                ))}
            </div>
            {isExpanded && Detail && row !== undefined && (
                <div
                    className="sticky left-0 border-hairline border-b bg-surface-1"
                    style={{ width: viewportWidth, height: detailHeight }}
                >
                    <div className="h-full overflow-hidden bg-surface-2 pt-3 pr-4 pb-4 pl-11">
                        <div className="overflow-hidden rounded-[10px] border border-hairline bg-canvas">
                            <Detail row={row} />
                        </div>
                    </div>
                </div>
            )}
        </div>
    ),
) as <T>(props: DataTableRowProps<T>) => ReactNode

/**
 * Cell box: a 1px transparent border and 13px padding around a single line,
 * the metrics AG Grid's quartz theme gave every cell.
 */
const cellClass = 'h-full shrink-0 overflow-hidden text-ellipsis whitespace-nowrap border border-transparent px-3.25'

/** Cell body of a loaded row: the column's renderer, or its raw field value. */
const renderCell = <T,>(column: DataTableColumn<T>, row: T): ReactNode => {
    if (column.render) {
        return column.render(row)
    }

    return column.field ? ((row[column.field] as ReactNode) ?? null) : null
}
