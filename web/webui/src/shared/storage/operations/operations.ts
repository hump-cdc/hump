import type { EventModel, RetryEventModel } from '@/shared/api'

import { db } from '../db'
import { deriveOperationStatus } from './derive-operation-status'
import type { OperationDto, OperationsFilterCriteria } from './types'

/** Ring-buffer cap: the store keeps at most this many newest operations. */
export const OPERATIONS_MAX = 10_000

/** Trim the store back to `OPERATIONS_MAX`, dropping the oldest operations. */
export const pruneOperations = async (): Promise<void> => {
    const total = await db.operations.count()
    const excess = total - OPERATIONS_MAX
    if (excess <= 0) return

    const stale = await db.operations.orderBy('timestamp').limit(excess).primaryKeys()
    await db.operations.bulkDelete(stale)
}

/** Fields searched by `OperationsFilterCriteria.search` (object fields contribute their values only). */
const searchHaystack = (record: OperationDto): string =>
    [
        record.operation,
        record.sinkType,
        record.schema,
        record.table,
        record.target,
        record.host,
        record.payload ?? '',
        ...Object.values(record.metadata ?? {}),
        ...Object.values(record.headers ?? {}),
    ]
        .join('\n')
        .toLowerCase()

/** Build a record predicate from filter criteria (strict facets, AND-ed search words). */
export const buildOperationsPredicate = (filter: OperationsFilterCriteria): ((record: OperationDto) => boolean) => {
    const words = filter.search.toLowerCase().split(/\s+/).filter(Boolean)

    return (record) => {
        if (!filter.sinks.includes(record.sinkType)) return false
        if (!filter.tables.includes(`${record.schema}.${record.table}`)) return false
        if (!filter.operations.includes(record.operation)) return false
        if (!filter.statuses.includes(deriveOperationStatus(record))) return false
        if (words.length === 0) return true

        const haystack = searchHaystack(record)
        return words.every((word) => haystack.includes(word))
    }
}

/**
 * Read a newest-first window of operations matching `filter`; the signature matches the data
 * table's `fetchRows`. `total` is the filtered count, sizing the table's scrollbar.
 */
export const getOperations = async ({
    startRow,
    endRow,
    filter,
}: {
    startRow: number
    endRow: number
    filter: OperationsFilterCriteria
}): Promise<{ rows: OperationDto[]; total: number }> => {
    const predicate = buildOperationsPredicate(filter)
    // Collections are single-use, so build a fresh one per query.
    const matching = () => db.operations.orderBy('timestamp').reverse().filter(predicate)

    const [total, rows] = await Promise.all([
        matching().count(),
        matching()
            .offset(startRow)
            .limit(Math.max(0, endRow - startRow))
            .toArray(),
    ])
    return { rows, total }
}

/** One operation's contribution to a batch, keyed by operation id in {@link saveOperationBatch}. */
export type OperationBatchEntry = {
    /** The operation event; absent when only retries for an already-stored operation arrived. */
    operation?: EventModel
    /** Retry attempts observed for the operation, in arrival order. */
    retries: RetryEventModel[]
}

/**
 * Persist a batch of operations and their retries in one transaction, upserting by `id` and
 * keeping retries that were stored earlier. An entry carrying only retries is a logged no-op
 * when its operation is not stored — the ring buffer has already dropped it.
 */
export const saveOperationBatch = async (batch: ReadonlyMap<string, OperationBatchEntry>): Promise<void> => {
    if (batch.size === 0) return

    const entries = [...batch]
    await db.transaction('rw', db.operations, async () => {
        const stored = await db.operations.bulkGet(entries.map(([id]) => id))
        const records: OperationDto[] = []

        entries.forEach(([id, entry], index) => {
            const existing = stored[index]
            const base = entry.operation ?? existing
            if (!base) {
                console.warn('[storage] retry for unknown operation', id)
                return
            }

            const retries = [...(existing?.retries ?? []), ...entry.retries]
            records.push(retries.length > 0 ? { ...base, retries } : base)
        })

        await db.operations.bulkPut(records)
    })
    await pruneOperations()
}

/** Remove every stored operation. */
export const clearOperations = async (): Promise<void> => {
    await db.operations.clear()
}
