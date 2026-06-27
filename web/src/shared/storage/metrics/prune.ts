import { METRIC_SAMPLES_MAX } from './constants'
import { SCALAR_BUCKETS, SINK_DELIVERY_BUCKETS, SINK_LATENCY_BUCKETS } from './schema'
import type { Bucket } from './types'

/** Erase a bucket store's types, keeping only what retention needs: its window size and a delete-below. */
const asPrunable = <T, TKey>({ table, sizeMs }: Bucket<T, TKey>) => ({
    sizeMs,
    deleteBelow: (cutoff: number): Promise<number> => table().where('timestamp').below(cutoff).delete(),
})

/** Every bucket store subject to retention, regardless of which series it holds. */
const ALL_BUCKETS = [
    ...SCALAR_BUCKETS.map(asPrunable),
    ...SINK_DELIVERY_BUCKETS.map(asPrunable),
    ...SINK_LATENCY_BUCKETS.map(asPrunable),
]

/** Drop samples older than `METRIC_SAMPLES_MAX` windows (e.g. the 5m bucket keeps the last 5m × MAX). */
export const pruneStaleMetrics = async (now: number): Promise<void> => {
    for (const { deleteBelow, sizeMs } of ALL_BUCKETS) {
        await deleteBelow(now - sizeMs * METRIC_SAMPLES_MAX)
    }
}
