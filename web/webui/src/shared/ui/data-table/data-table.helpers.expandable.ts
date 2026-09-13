import { useCallback, useEffect, useMemo, useRef, useState } from 'react'

/** Internal row wrapper: real master rows, injected detail rows, and not-yet-loaded placeholders. */
export type InternalRow<T> =
    | { kind: 'master'; id: string; index: number; row: T; expandable: boolean; expanded: boolean }
    | { kind: 'detail'; id: string; parentId: string; row: T }
    | { kind: 'placeholder'; id: string; index: number }

/** Shared via the grid `context` so the expand cell can toggle a row. */
export type ExpandContext = {
    toggle: (id: string) => void
}

const DETAIL_SUFFIX = '::detail'
const PLACEHOLDER_PREFIX = '__placeholder:'

/** Builds an expanded set toggler backed by React state. */
const useExpandedSet = () => {
    const [expanded, setExpanded] = useState<ReadonlySet<string>>(() => new Set())
    const toggle = useCallback((id: string) => {
        setExpanded((prev) => {
            const next = new Set(prev)
            if (next.has(id)) {
                next.delete(id)
            } else {
                next.add(id)
            }
            return next
        })
    }, [])
    return { expanded, toggle }
}

/** Appends a master row (and its detail row when expanded) to `out`. */
const pushMaster = <T>(
    out: InternalRow<T>[],
    row: T,
    index: number,
    id: string,
    expandable: boolean,
    expanded: ReadonlySet<string>,
): void => {
    const isExpanded = expandable && expanded.has(id)
    out.push({ kind: 'master', id, index, row, expandable, expanded: isExpanded })
    if (isExpanded) {
        out.push({ kind: 'detail', id: `${id}${DETAIL_SUFFIX}`, parentId: id, row })
    }
}

type ClientOptions<T> = {
    data: T[]
    getRowId: (row: T) => string
    expandable: boolean
    isRowExpandable: (row: T) => boolean
}

/**
 * Client mode: flattens in-memory `data` into master rows, injecting a detail
 * row after each expanded master.
 */
export const useExpandableRows = <T>({ data, getRowId, expandable, isRowExpandable }: ClientOptions<T>) => {
    const { expanded, toggle } = useExpandedSet()

    const rows = useMemo<InternalRow<T>[]>(() => {
        const result: InternalRow<T>[] = []
        data.forEach((row, index) => {
            pushMaster(result, row, index, getRowId(row), expandable && isRowExpandable(row), expanded)
        })
        return result
    }, [data, getRowId, expandable, isRowExpandable, expanded])

    return { rows, toggle }
}

type PrefixOptions<T> = {
    data: T[]
    rowCount: number
    getRowId: (row: T) => string
    expandable: boolean
    isRowExpandable: (row: T) => boolean
}

/**
 * Infinite mode: flattens the contiguous loaded prefix `data` into master rows
 * (injecting detail rows after expanded masters) and pads the tail with
 * placeholders for indices `data.length .. rowCount-1`, so the scrollbar is
 * stable and unloaded rows render as skeletons.
 */
export const usePrefixRows = <T>({ data, rowCount, getRowId, expandable, isRowExpandable }: PrefixOptions<T>) => {
    const { expanded, toggle } = useExpandedSet()

    const rows = useMemo<InternalRow<T>[]>(() => {
        const result: InternalRow<T>[] = []
        data.forEach((row, index) => {
            pushMaster(result, row, index, getRowId(row), expandable && isRowExpandable(row), expanded)
        })
        for (let index = data.length; index < rowCount; index++) {
            result.push({ kind: 'placeholder', id: `${PLACEHOLDER_PREFIX}${index}`, index })
        }
        return result
    }, [data, rowCount, getRowId, expandable, isRowExpandable, expanded])

    return { rows, toggle }
}

type WindowedOptions<T> = {
    rowCount?: number
    blockSize: number
    fetchRows: (params: { startRow: number; endRow: number }) => Promise<{ rows: T[]; total?: number }>
    getRowId: (row: T) => string
    expandable: boolean
    isRowExpandable: (row: T) => boolean
    /** When this changes, cached blocks are dropped and the visible window is refetched in place. */
    refreshToken?: number | string
}

/**
 * Server mode: keeps a sparse map of loaded rows by index. When `rowCount` is
 * known the list is padded with placeholders (stable scrollbar); otherwise it
 * grows as blocks load. `ensureRange` fetches the blocks covering a row-index
 * window on demand.
 */
export const useWindowedRows = <T>({
    rowCount,
    blockSize,
    fetchRows,
    getRowId,
    expandable,
    isRowExpandable,
    refreshToken,
}: WindowedOptions<T>) => {
    const { expanded, toggle } = useExpandedSet()
    const [loaded, setLoaded] = useState<ReadonlyMap<number, T>>(() => new Map())
    const loadedCountRef = useRef(0)
    const loadingBlocks = useRef<Set<number>>(new Set())
    const doneRef = useRef(false)
    const lastRangeRef = useRef({ start: 0, end: blockSize })

    const loadBlock = useCallback(
        (block: number) => {
            if (loadingBlocks.current.has(block)) {
                return
            }
            loadingBlocks.current.add(block)
            const startRow = block * blockSize
            fetchRows({ startRow, endRow: startRow + blockSize })
                .then(({ rows, total }) => {
                    if (rows.length < blockSize) {
                        doneRef.current = true
                    }
                    setLoaded((prev) => {
                        const next = new Map(prev)
                        rows.forEach((row, offset) => {
                            next.set(startRow + offset, row)
                        })
                        loadedCountRef.current = next.size
                        if (total != null && next.size >= total) {
                            doneRef.current = true
                        }
                        return next
                    })
                })
                .catch(() => {
                    // Leave the block unloaded; a later scroll can retry it.
                })
                .finally(() => {
                    loadingBlocks.current.delete(block)
                })
        },
        [blockSize, fetchRows],
    )

    const ensureRange = useCallback(
        (startIndex: number, endIndex: number) => {
            lastRangeRef.current = { start: startIndex, end: endIndex }
            const firstBlock = Math.max(0, Math.floor(startIndex / blockSize))
            const lastBlock = Math.floor(endIndex / blockSize)
            for (let block = firstBlock; block <= lastBlock; block++) {
                if (!loaded.has(block * blockSize)) {
                    loadBlock(block)
                }
            }
            // Unknown total: pull the next block once the tail comes into view.
            if (rowCount == null && !doneRef.current && endIndex >= loadedCountRef.current - 1) {
                loadBlock(Math.floor(loadedCountRef.current / blockSize))
            }
        },
        [blockSize, loaded, loadBlock, rowCount],
    )

    // Drop cached blocks and refetch the visible window when `refreshToken` changes.
    // Used by live data sources whose row order shifts (e.g. new rows at the top).
    const reload = useCallback(() => {
        loadingBlocks.current.clear()
        doneRef.current = false
        loadedCountRef.current = 0
        setLoaded(new Map())
        const { start, end } = lastRangeRef.current
        const firstBlock = Math.max(0, Math.floor(start / blockSize))
        const lastBlock = Math.floor(end / blockSize)
        for (let block = firstBlock; block <= lastBlock; block++) {
            loadBlock(block)
        }
    }, [blockSize, loadBlock])

    const reloadRef = useRef(reload)
    reloadRef.current = reload
    const firstRun = useRef(true)
    // biome-ignore lint/correctness/useExhaustiveDependencies: refreshToken is an intentional trigger; reload is read via a ref.
    useEffect(() => {
        if (firstRun.current) {
            firstRun.current = false
            return
        }
        reloadRef.current()
    }, [refreshToken])

    const rows = useMemo<InternalRow<T>[]>(() => {
        const total = rowCount ?? loaded.size
        const result: InternalRow<T>[] = []
        for (let index = 0; index < total; index++) {
            const row = loaded.get(index)
            if (!row) {
                result.push({ kind: 'placeholder', id: `${PLACEHOLDER_PREFIX}${index}`, index })
                continue
            }
            pushMaster(result, row, index, getRowId(row), expandable && isRowExpandable(row), expanded)
        }
        return result
    }, [rowCount, loaded, getRowId, expandable, isRowExpandable, expanded])

    return { rows, toggle, ensureRange }
}
