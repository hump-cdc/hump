import { useCallback, useEffect, useMemo, useRef, useState } from 'react'

import { useLiveQuery } from 'dexie-react-hooks'
import { useShallow } from 'zustand/react/shallow'

import { useOperationsFilter } from '@/entities/operations/filter/@x/list'
import { db, deriveOperationStatus, getOperations, type OperationDto } from '@/shared/storage'
import { useDebounce } from '@/shared/utils'

/** Rows fetched per block from IndexedDB (server-mode window size). */
const BLOCK_SIZE = 100
/** Debounce (ms) for refetching after IndexedDB changes, to absorb stream bursts. */
const REFETCH_DEBOUNCE_MS = 300

export type Attempt = {
    attempt: number
    started: Date
    durationMs: number
    result: 'failed' | 'delivered'
    error?: string
}

export type Operation = {
    id: string
    time: Date
    table: string
    op: 'insert' | 'update' | 'delete' | 'select'
    sink: string
    sinkHost: string
    status: 'delivered' | 'failed' | 'retrying'
    attempts: number
    durationMs: number
    error?: string
    retries?: Attempt[]
}
/**
 * Live operations feed for {@link OperationsList}, shaped for the DataTable
 * server (windowed) mode: any viewport window is fetched on demand by row range,
 * and `refreshToken` invalidates cached blocks when the store changes.
 */
export const useOperations = () => {
    const filter = useOperationsFilter(
        useShallow((state) => ({
            sinks: state.sinks,
            tables: state.tables,
            operations: state.operations,
            statuses: state.statuses,
            search: state.search,
        })),
    )
    const filterKey = useMemo(() => JSON.stringify(filter), [filter])

    const fetchRows = useCallback(
        async ({ startRow, endRow }: { startRow: number; endRow: number }) => {
            const { rows, total } = await getOperations({ startRow, endRow, filter })
            return { rows: rows.map(mapOperation), total }
        },
        [filter],
    )

    // Live store fingerprint: total count (sizes the scrollbar) + newest id (drives refetch).
    const meta = useLiveQuery(async () => {
        const [count, latest] = await Promise.all([
            db.operations.count(),
            db.operations.orderBy('timestamp').reverse().limit(1).toArray(),
        ])
        return { count, token: `${count}:${latest[0]?.id ?? ''}` }
    })
    const token = meta?.token

    // Debounced fingerprint: invalidates the table's cached blocks on store changes,
    // coalescing stream bursts so the visible window refetches at most ~once per window.
    const [storeToken, setStoreToken] = useState<string>()
    const tokenRef = useRef(token)
    tokenRef.current = token
    const bump = useDebounce(() => setStoreToken(tokenRef.current), REFETCH_DEBOUNCE_MS)

    useEffect(() => {
        if (!token) return
        bump()
        return () => bump.cancel()
    }, [token, bump])

    // Filter changes invalidate immediately (no debounce); store bursts stay debounced.
    const refreshToken = `${storeToken ?? ''}|${filterKey}`

    return {
        fetchRows,
        rowCount: meta?.count ?? 0,
        blockSize: BLOCK_SIZE,
        refreshToken,
    } as const
}

const mapAttempt = (
    attempt: number,
    timestamp: string,
    durationMs: number,
    success: boolean,
    error?: string,
): Attempt => ({
    attempt,
    started: new Date(timestamp),
    durationMs,
    result: success ? 'delivered' : 'failed',
    error,
})

export const mapOperation = (record: OperationDto): Operation => {
    const attempts: Attempt[] = [
        mapAttempt(1, record.timestamp, record.durationMs, record.success, record.error),
        ...(record.retries ?? []).map((retry) =>
            mapAttempt(retry.attempt, retry.timestamp, retry.durationMs, retry.success, retry.error),
        ),
    ]
    const final = attempts[attempts.length - 1]

    return {
        id: record.id,
        time: new Date(record.timestamp),
        table: record.schema ? `${record.schema}.${record.table}` : record.table,
        op: record.operation as Operation['op'],
        sink: record.sinkType,
        sinkHost: record.host,
        status: deriveOperationStatus(record),
        attempts: attempts.length,
        durationMs: final.durationMs,
        error: final.result === 'failed' ? final.error : undefined,
        retries: attempts.length > 1 ? attempts : undefined,
    }
}
