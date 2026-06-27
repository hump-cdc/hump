import type { EventModel, RetryEventModel } from '@/shared/api'
import {
    type FlowOperationDelta,
    type OperationBatchEntry,
    saveOperationBatch,
    saveOperationDeltas,
} from '@/shared/storage'

/**
 * How long events accumulate before being written. Per-event transactions would make
 * IndexedDB the bottleneck under a burst, and wake every watching `liveQuery` as often.
 */
const FLUSH_INTERVAL_MS = 500

/**
 * Buffers the operations stream on a single timer, in the two shapes the stores want: rows
 * keyed by operation id, and flow-graph counts folded by the edge each event travelled.
 *
 * The two halves are written in separate transactions: they touch disjoint stores, and a
 * failed count must not roll back the rows it was counting.
 */
export class StreamBuffer {
    readonly #operations = new Map<string, OperationBatchEntry>()
    readonly #flows = new Map<string, FlowOperationDelta>()
    #timer: ReturnType<typeof setTimeout> | null = null

    /** Buffer a delivered operation as a row and as a count on its edge. */
    add(event: EventModel): void {
        this.#entry(event.id).operation = event
        this.#count(event)
        this.#schedule()
    }

    /** Buffer a retry against its parent operation, pending or already stored. */
    addRetry(retry: RetryEventModel): void {
        this.#entry(retry.operationId).retries.push(retry)
        this.#schedule()
    }

    /** Write everything buffered so far. Safe to call when nothing is pending. */
    async flush(): Promise<void> {
        await Promise.all([this.#flushOperations(), this.#flushFlows()])
    }

    /**
     * Drop the pending flush timer and unwind. Buffered counts describe a stream this worker
     * no longer consumes, so they go; the operations behind them get one last write.
     */
    async stop(): Promise<void> {
        if (this.#timer !== null) clearTimeout(this.#timer)
        this.#timer = null
        this.#flows.clear()
        await this.#flushOperations()
    }

    /** The row entry for `id`, created empty on the first event mentioning it. */
    #entry(id: string): OperationBatchEntry {
        const existing = this.#operations.get(id)
        if (existing) return existing

        const entry: OperationBatchEntry = { retries: [] }
        this.#operations.set(id, entry)
        return entry
    }

    /** Fold one operation into the count for the edge it travelled. */
    #count(event: EventModel): void {
        const key = [event.schema, event.table, event.sinkType, event.host, event.target].join(' ')
        const existing = this.#flows.get(key)

        if (existing) existing.count += 1
        else
            this.#flows.set(key, {
                schema: event.schema,
                table: event.table,
                sinkType: event.sinkType,
                host: event.host,
                target: event.target,
                count: 1,
            })
    }

    async #flushOperations(): Promise<void> {
        if (this.#operations.size === 0) return

        const batch = new Map(this.#operations)
        this.#operations.clear()

        try {
            await saveOperationBatch(batch)
        } catch (error) {
            console.error('[storage:write]', error)
        }
    }

    async #flushFlows(): Promise<void> {
        if (this.#flows.size === 0) return

        const deltas = [...this.#flows.values()]
        this.#flows.clear()

        try {
            await saveOperationDeltas(deltas)
        } catch (error) {
            console.error('[storage:flow]', error)
        }
    }

    /** Arm the flush timer unless one is already pending. */
    #schedule(): void {
        this.#timer ??= setTimeout(() => {
            this.#timer = null
            void this.flush()
        }, FLUSH_INTERVAL_MS)
    }
}
