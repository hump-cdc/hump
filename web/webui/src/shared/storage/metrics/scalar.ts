import { db } from '../db'
import { METRIC_BUCKET_MS, METRIC_SAMPLES_MAX } from './constants'
import { floorToBucket, SCALAR_BUCKETS, SCALAR_INDEX, SCALAR_TABLES } from './schema'
import type { MetricBucket, MetricDto, ScalarMetricName } from './types'
import { type Aggregation, fold } from './utils/fold'
import { groupOf, pruneSamples } from './utils/groups'
import { normalize } from './utils/normalize'

/**
 * Read one scalar series in the given bucket window, chart-ready: the newest `METRIC_SAMPLES_MAX`
 * samples oldest → newest, zero-padded to a fixed count and stripped down to `timestamp`/`value`.
 */
export const getScalar = async (metric: ScalarMetricName, bucket: MetricBucket = '1m'): Promise<MetricDto[]> => {
    const items = await groupOf(SCALAR_TABLES[bucket](), SCALAR_INDEX, metric)
        .reverse()
        .limit(METRIC_SAMPLES_MAX)
        .toArray()

    return normalize(items, METRIC_BUCKET_MS[bucket])
}

/**
 * Record a scalar observation into the 1m/5m/10m stores, folding it per `aggregate` into whatever
 * `[metric+timestamp]` already holds. Each series is then trimmed to `METRIC_SAMPLES_MAX`.
 */
export const saveScalar = async (
    metric: ScalarMetricName,
    value: number,
    now: number,
    aggregate: Aggregation = 'max',
): Promise<void> => {
    for (const { table: getTable, sizeMs } of SCALAR_BUCKETS) {
        const table = getTable()
        const timestamp = floorToBucket(now, sizeMs)

        await db.transaction('rw', table, async () => {
            const existing = await table.get([metric, timestamp])
            await table.put({ metric, timestamp, value: fold(existing, value, aggregate) })
        })

        await pruneSamples(table, SCALAR_INDEX, metric)
    }
}
