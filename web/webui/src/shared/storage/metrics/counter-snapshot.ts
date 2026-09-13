import { getMeta, saveMeta } from '../meta/meta'

/** Key the counter snapshot is persisted under in the `meta` store. */
const COUNTER_SNAPSHOT_KEY = 'metricCounters'

/** The previous poll's cumulative counter readings and the exporter they came from. */
export type CounterSnapshotDto = {
    /** Exporter `process_start_time_seconds`; a changed value means every counter reset. */
    startedAt: number | undefined
    /** Last-seen cumulative reading per counter series, keyed by series identity. */
    counters: Record<string, number>
}

const EMPTY_COUNTER_SNAPSHOT: CounterSnapshotDto = { startedAt: undefined, counters: {} }

/**
 * Read the baseline the next poll's deltas are measured against. Unreadable or outdated state
 * yields an empty snapshot, costing one poll.
 */
export const getCounterSnapshot = async (): Promise<CounterSnapshotDto> => {
    const stored = await getMeta(COUNTER_SNAPSHOT_KEY)
    if (!stored) return EMPTY_COUNTER_SNAPSHOT

    try {
        const parsed: unknown = JSON.parse(stored)
        if (typeof parsed !== 'object' || parsed === null) return EMPTY_COUNTER_SNAPSHOT

        const { startedAt, counters } = parsed as Partial<CounterSnapshotDto>
        if (typeof counters !== 'object' || counters === null) return EMPTY_COUNTER_SNAPSHOT

        return { startedAt: typeof startedAt === 'number' ? startedAt : undefined, counters }
    } catch {
        return EMPTY_COUNTER_SNAPSHOT
    }
}

/** Persist the baseline for the next poll's deltas. */
export const saveCounterSnapshot = (snapshot: CounterSnapshotDto): Promise<string> =>
    saveMeta(COUNTER_SNAPSHOT_KEY, JSON.stringify(snapshot))
