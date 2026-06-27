import { db } from '../db'
import { floorToBucket, SINK_INDEX, SINK_LATENCY_BUCKETS, SINK_LATENCY_TABLES } from './schema'
import type { MetricBucket, SinkMetricDto } from './types'
import { fold } from './utils/fold'
import { pruneSamples } from './utils/groups'

/**
 * Read every stored per-sink latency sample in the given bucket window, across all sinks. Each
 * sink keeps at most `METRIC_SAMPLES_MAX` samples, so the whole store is already chart-sized.
 */
export const getSinkLatency = (bucket: MetricBucket = '1m'): Promise<SinkMetricDto[]> =>
    SINK_LATENCY_TABLES[bucket]().toArray()

/**
 * Record a latency observation for `sink` into the 1m/5m/10m bucket stores, keeping the peak
 * observed in each window. Every sink keeps its own newest `METRIC_SAMPLES_MAX` samples.
 */
export const saveSinkLatency = async (sink: string, value: number, now: number): Promise<void> => {
    for (const { table: getTable, sizeMs } of SINK_LATENCY_BUCKETS) {
        const table = getTable()
        const timestamp = floorToBucket(now, sizeMs)

        await db.transaction('rw', table, async () => {
            const existing = await table.get([sink, timestamp])
            await table.put({ sink, timestamp, value: fold(existing, value, 'max') })
        })

        await pruneSamples(table, SINK_INDEX, sink)
    }
}
