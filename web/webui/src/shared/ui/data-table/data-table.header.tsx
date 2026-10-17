import type { ReactNode, Ref } from 'react'

import { cn } from '@/shared/utils'

import type { ColumnHeader } from './data-table.column-model'
import { EXPAND_COLUMN_WIDTH } from './data-table.consts'
import type { DataTableColumn } from './data-table.types'

type DataTableHeaderProps<T> = {
    columns: DataTableColumn<T>[]
    headers: ColumnHeader[]
    widths: number[]
    totalWidth: number
    height: number
    isExpandable: boolean
    /** Right padding matching the body's vertical scrollbar, so the columns line up. */
    scrollbarWidth: number
    ref: Ref<HTMLDivElement>
}

/**
 * Header row, kept outside the scrolling body so it stays put; the table
 * mirrors the body's horizontal scroll onto it through `ref`.
 */
export const DataTableHeader = <T,>({
    columns,
    headers,
    widths,
    totalWidth,
    height,
    isExpandable,
    scrollbarWidth,
    ref,
}: DataTableHeaderProps<T>): ReactNode => (
    <div
        ref={ref}
        className="shrink-0 overflow-hidden border-hairline border-b bg-surface-2"
        style={{ height: height + 1, paddingRight: scrollbarWidth }}
    >
        <div className="flex h-full" style={{ width: totalWidth }}>
            {isExpandable && <div className="shrink-0" style={{ width: EXPAND_COLUMN_WIDTH }} />}
            {headers.map((header, index) => (
                <HeaderCell key={header.id} column={columns[index]} header={header} width={widths[index]} />
            ))}
        </div>
    </div>
)

type HeaderCellProps<T> = {
    column: DataTableColumn<T>
    header: ColumnHeader
    width: number
}

/** One column title with the resize grip on its right edge. */
const HeaderCell = <T,>({ column, header, width }: HeaderCellProps<T>): ReactNode => (
    <div
        className={cn(
            'relative flex h-full shrink-0 items-center px-3.5 font-semibold text-ink-subtle text-xs uppercase tracking-caps',
            column.headerClass,
        )}
        style={{ width }}
    >
        <span className="truncate">{column.header}</span>
        {column.resizable !== false && <ResizeGrip header={header} />}
    </div>
)

/**
 * Drag handle straddling the column's right border, drawn as a short divider.
 * Flex columns show the divider too but refill the width, so they get no handler.
 */
const ResizeGrip = ({ header }: { header: ColumnHeader }): ReactNode => {
    const canResize = header.column.getCanResize()
    const onResize = canResize ? header.getResizeHandler() : undefined

    return (
        <div
            aria-hidden
            onMouseDown={onResize}
            onTouchStart={onResize}
            className={cn(
                'absolute top-0 -right-0.75 z-1 h-full w-2 touch-none select-none',
                "after:absolute after:top-[35%] after:left-0.5 after:h-[30%] after:w-0.5 after:bg-hairline after:content-['']",
                canResize ? 'cursor-ew-resize' : 'cursor-default',
            )}
        />
    )
}
