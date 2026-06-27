import { db } from '../db'
import { floorToBucket, SINK_DELIVERY_BUCKETS, SINK_DELIVERY_TABLES, SINK_INDEX } from './schema'
import type { MetricBucket, SinkMetricDto } from './types'
import { fold } from './utils/fold'
import { pruneSamples } from './utils/groups'

/**
 * Read every stored per-sink delivery sample in the given bucket window, across all sinks. Each
 * sink keeps at most `METRIC_SAMPLES_MAX` samples, so the whole store is already chart-sized.
 */
export const getSinkDelivery = (bucket: MetricBucket = '1m'): Promise<SinkMetricDto[]> =>
    SINK_DELIVERY_TABLES[bucket]().toArray()

/**
 * Record the messages `sink` delivered since the previous poll into the 1m/5m/10m stores. The
 * value is a per-poll delta, so repeated observations in one window sum. Every sink keeps its own
 * newest `METRIC_SAMPLES_MAX` samples.
 */
export const saveSinkDelivery = async (sink: string, value: number, now: number): Promise<void> => {
    for (const { table: getTable, sizeMs } of SINK_DELIVERY_BUCKETS) {
        const table = getTable()
        const timestamp = floorToBucket(now, sizeMs)

        await db.transaction('rw', table, async () => {
            const existing = await table.get([sink, timestamp])
            await table.put({ sink, timestamp, value: fold(existing, value, 'sum') })
        })

        await pruneSamples(table, SINK_INDEX, sink)
    }
}
