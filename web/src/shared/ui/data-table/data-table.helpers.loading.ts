import { useCallback, useEffect, useRef } from 'react'

import type { ViewportChangedEvent } from 'ag-grid-community'

import { SCROLL_LOAD_THRESHOLD } from './data-table.consts'
import type { InternalRow } from './data-table.helpers.expandable'

/**
 * Builds a viewport handler that loads the data blocks overlapping the visible
 * rows (plus a slack threshold) via `ensureRange`.
 */
export const useViewportRangeLoader = <T>(ensureRange: (start: number, end: number) => void) =>
    useCallback(
        (event: ViewportChangedEvent<InternalRow<T>>) => {
            const { api } = event
            const first = api.getFirstDisplayedRowIndex()
            const last = api.getLastDisplayedRowIndex()
            if (first < 0 || last < 0) {
                return
            }
            let min = Number.POSITIVE_INFINITY
            let max = -1
            for (let i = first; i <= last; i++) {
                const data = api.getDisplayedRowAtIndex(i)?.data
                if (data && (data.kind === 'master' || data.kind === 'placeholder')) {
                    min = Math.min(min, data.index)
                    max = Math.max(max, data.index)
                }
            }
            if (max < 0) {
                return
            }
            ensureRange(min - SCROLL_LOAD_THRESHOLD, max + SCROLL_LOAD_THRESHOLD)
        },
        [ensureRange],
    )

/** Kicks off the first block once so an unknown-total list has something to render. */
export const useInitialBlockLoad = (ensureRange: (start: number, end: number) => void, blockSize: number) => {
    const initializedRef = useRef(false)
    useEffect(() => {
        if (initializedRef.current) {
            return
        }
        initializedRef.current = true
        ensureRange(0, blockSize)
    }, [ensureRange, blockSize])
}
