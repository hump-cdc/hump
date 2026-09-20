import {
    type ColumnDef,
    type ColumnSizingState,
    getCoreRowModel,
    type OnChangeFn,
    type Table,
    useReactTable,
} from '@tanstack/react-table'
import { useVirtualizer, type VirtualItem } from '@tanstack/react-virtual'

import {
    type ReactElement,
    type RefObject,
    type UIEvent,
    useCallback,
    useEffect,
    useLayoutEffect,
    useMemo,
    useRef,
    useState,
} from 'react'

import { cn } from '@/shared/utils'

import { loadColumnSizing, saveColumnSizing } from './data-table.column-state'
import {
    DEFAULT_DETAIL_HEIGHT,
    DEFAULT_HEADER_HEIGHT,
    DEFAULT_OVERSCAN,
    DEFAULT_ROW_HEIGHT,
    EXPAND_COLUMN_WIDTH,
    LOADING_ROW_COUNT,
} from './data-table.consts'
import { DataTableHeader } from './data-table.header'
import { DataTableRow } from './data-table.row'
import { type ColumnSizing, columnId, isResizableColumn, resolveColumnWidths } from './data-table.sizing'
import type { DataTableColumn, DataTableProps, DataTableRange } from './data-table.types'

/** Stand-in row data for TanStack Table, which here only models the columns. */
const NO_ROWS: never[] = []

/**
 * A virtualized table with a fixed header, resizable columns and expandable
 * detail rows, built on TanStack Table (columns) and TanStack Virtual (rows).
 * It renders `rowCount` rows by index and never fetches: unloaded rows show
 * skeletons until `getRow` returns them.
 */
export const DataTable = <T,>({
    columns,
    id,
    getRowId,
    rowCount,
    getRow,
    onVisibleRangeChange,
    isLoading = false,
    getDetail,
    isRowExpandable,
    detailHeight = DEFAULT_DETAIL_HEIGHT,
    rowHeight = DEFAULT_ROW_HEIGHT,
    headerHeight = DEFAULT_HEADER_HEIGHT,
    overscan = DEFAULT_OVERSCAN,
    className,
    height,
    emptyMessage = 'No Rows To Show',
}: DataTableProps<T>): ReactElement => {
    const bodyRef = useRef<HTMLDivElement>(null)
    const headerRef = useRef<HTMLDivElement>(null)
    const viewport = useViewportSize(bodyRef)
    const isExpandable = !!getDetail

    // 1. Column widths: persisted resizes, then flex columns fill what is left.
    const [columnSizing, setColumnSizing] = usePersistedSizing(id, columns)
    const expandWidth = isExpandable ? EXPAND_COLUMN_WIDTH : 0
    const widths = useMemo(
        () => resolveColumnWidths(columns, columnSizing, viewport.width - expandWidth),
        [columns, columnSizing, viewport.width, expandWidth],
    )
    const totalWidth = useMemo(() => widths.reduce((sum, width) => sum + width, expandWidth), [widths, expandWidth])
    const table = useColumnModel(columns, widths, columnSizing, setColumnSizing)
    usePersistOnResizeEnd(table, id, columnSizing)

    // 2. Rows: expansion follows row ids, so it survives rows shifting index.
    const [expanded, toggle] = useExpandedRows()
    const canExpand = useCallback(
        (row: T) => isExpandable && (isRowExpandable ? isRowExpandable(row) : true),
        [isExpandable, isRowExpandable],
    )
    const detailHeightOf = useCallback(
        (row: T) => (typeof detailHeight === 'function' ? detailHeight(row) : detailHeight),
        [detailHeight],
    )
    const isRowExpanded = useCallback(
        (row: T | undefined) => row !== undefined && canExpand(row) && expanded.has(getRowId(row)),
        [canExpand, expanded, getRowId],
    )

    // 3. Virtualization: sizes are exact, so no DOM measuring is needed.
    const count = rowCount === 0 && isLoading ? LOADING_ROW_COUNT : rowCount
    const rowAt = useCallback((index: number) => (rowCount === 0 ? undefined : getRow(index)), [rowCount, getRow])
    const sizeAt = useCallback(
        (index: number) => {
            const row = rowAt(index)
            return isRowExpanded(row) && row !== undefined ? rowHeight + detailHeightOf(row) : rowHeight
        },
        [rowAt, isRowExpanded, rowHeight, detailHeightOf],
    )
    const virtualizer = useVirtualizer({
        count,
        getScrollElement: () => bodyRef.current,
        estimateSize: sizeAt,
        overscan,
    })
    // biome-ignore lint/correctness/useExhaustiveDependencies: a new `sizeAt` means cached sizes are stale.
    useLayoutEffect(() => {
        virtualizer.measure()
    }, [virtualizer, sizeAt])

    const items = virtualizer.getVirtualItems()
    useRangeReport(items, rowCount, onVisibleRangeChange)

    const syncHeaderScroll = useCallback((event: UIEvent<HTMLDivElement>) => {
        if (headerRef.current) {
            headerRef.current.scrollLeft = event.currentTarget.scrollLeft
        }
    }, [])

    return (
        <div
            className={cn(
                'flex h-full w-full flex-col overflow-hidden rounded-xl border border-hairline bg-surface-1 font-sans text-base text-ink leading-[normal]',
                className,
            )}
            style={height != null ? { height } : undefined}
        >
            <DataTableHeader
                ref={headerRef}
                columns={columns}
                headers={table.getFlatHeaders()}
                widths={widths}
                totalWidth={totalWidth}
                height={headerHeight}
                isExpandable={isExpandable}
                scrollbarWidth={viewport.scrollbarWidth}
            />
            {/* The scroller is taken out of flow so the rows never size the table or its parents. */}
            <div className="relative min-h-0 flex-1">
                <div
                    ref={bodyRef}
                    onScroll={syncHeaderScroll}
                    className="scrollbar-thin absolute inset-0 overflow-auto [scrollbar-color:var(--hairline-strong)_transparent]"
                >
                    {count === 0 ? (
                        <div className="flex h-full items-center justify-center">{emptyMessage}</div>
                    ) : (
                        <div className="relative" style={{ height: virtualizer.getTotalSize(), width: totalWidth }}>
                            {items.map((item) => {
                                const row = rowAt(item.index)
                                const isExpanded = isRowExpanded(row)
                                return (
                                    <DataTableRow<T>
                                        key={item.key}
                                        columns={columns}
                                        widths={widths}
                                        totalWidth={totalWidth}
                                        viewportWidth={viewport.width}
                                        row={row}
                                        rowId={row === undefined ? undefined : getRowId(row)}
                                        start={item.start}
                                        rowHeight={rowHeight}
                                        isExpandable={isExpandable}
                                        canExpand={row !== undefined && canExpand(row)}
                                        isExpanded={isExpanded}
                                        detailHeight={isExpanded && row !== undefined ? detailHeightOf(row) : 0}
                                        Detail={getDetail}
                                        onToggle={toggle}
                                    />
                                )
                            })}
                        </div>
                    )}
                </div>
            </div>
        </div>
    )
}

/** Inner width of the scrolling body and the width its vertical scrollbar takes. */
type ViewportSize = {
    width: number
    scrollbarWidth: number
}

/** Tracks the body's client width, which flex columns fill. */
const useViewportSize = (ref: RefObject<HTMLDivElement | null>): ViewportSize => {
    const [size, setSize] = useState<ViewportSize>({ width: 0, scrollbarWidth: 0 })

    useLayoutEffect(() => {
        const element = ref.current
        if (!element) {
            return
        }
        const update = () =>
            setSize((previous) => {
                const next = { width: element.clientWidth, scrollbarWidth: element.offsetWidth - element.clientWidth }
                return previous.width === next.width && previous.scrollbarWidth === next.scrollbarWidth
                    ? previous
                    : next
            })
        update()
        const observer = new ResizeObserver(update)
        observer.observe(element)
        return () => observer.disconnect()
    }, [ref])

    return size
}

/** Column widths the user dragged, seeded from localStorage when the table has an `id`. */
const usePersistedSizing = <T,>(
    id: string | undefined,
    columns: DataTableColumn<T>[],
): [ColumnSizing, OnChangeFn<ColumnSizingState>] => {
    const [sizing, setSizing] = useState<ColumnSizing>(() => (id ? loadColumnSizing(id, resizableIds(columns)) : {}))

    return [sizing, setSizing]
}

/**
 * TanStack Table over the columns only, driving the resize drag. Each column's
 * base size is its resolved width, so a drag starts from what is on screen.
 */
const useColumnModel = <T,>(
    columns: DataTableColumn<T>[],
    widths: number[],
    columnSizing: ColumnSizing,
    onColumnSizingChange: OnChangeFn<ColumnSizingState>,
): Table<T> => {
    const columnDefs = useMemo<ColumnDef<T>[]>(
        () =>
            columns.map((column, index) => ({
                id: columnId(index),
                size: widths[index],
                minSize: column.minWidth,
                maxSize: column.maxWidth,
                enableResizing: isResizableColumn(column),
            })),
        [columns, widths],
    )

    return useReactTable<T>({
        data: NO_ROWS,
        columns: columnDefs,
        getCoreRowModel: getCoreRowModel(),
        columnResizeMode: 'onChange',
        state: { columnSizing },
        onColumnSizingChange,
    })
}

/** Saves the column widths once a resize drag ends. */
const usePersistOnResizeEnd = <T,>(table: Table<T>, id: string | undefined, columnSizing: ColumnSizing): void => {
    const resizing = table.getState().columnSizingInfo.isResizingColumn
    const wasResizing = useRef(false)

    useEffect(() => {
        if (resizing) {
            wasResizing.current = true
            return
        }
        if (wasResizing.current && id) {
            saveColumnSizing(id, columnSizing)
        }
        wasResizing.current = false
    }, [resizing, id, columnSizing])
}

/** Ids of expanded rows and a toggle for one of them. */
const useExpandedRows = (): [ReadonlySet<string>, (id: string) => void] => {
    const [expanded, setExpanded] = useState<ReadonlySet<string>>(() => new Set())
    const toggle = useCallback(
        (id: string) =>
            setExpanded((previous) => {
                const next = new Set(previous)
                if (!next.delete(id)) {
                    next.add(id)
                }
                return next
            }),
        [],
    )

    return [expanded, toggle]
}

/** Reports the rendered index range whenever it changes; skeleton rows of an unknown count are skipped. */
const useRangeReport = (
    items: VirtualItem[],
    rowCount: number,
    onVisibleRangeChange: ((range: DataTableRange) => void) | undefined,
): void => {
    const start = items[0]?.index ?? 0
    const end = rowCount === 0 ? -1 : Math.min(items.at(-1)?.index ?? -1, rowCount - 1)

    useEffect(() => {
        if (end >= start) {
            onVisibleRangeChange?.({ start, end })
        }
    }, [start, end, onVisibleRangeChange])
}

/** Ids of the columns whose width the user may drag. */
const resizableIds = <T,>(columns: DataTableColumn<T>[]): string[] =>
    columns.flatMap((column, index) => (isResizableColumn(column) ? [columnId(index)] : []))
