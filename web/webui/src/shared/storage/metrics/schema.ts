import type { Table } from 'dexie'

import { db } from '../db'
import { METRIC_BUCKET_MS } from './constants'
import type { Bucket, MetricBucket, ScalarMetricDto, ScalarMetricName, SinkMetricDto } from './types'

/** Primary key of the scalar stores: the series name plus the rounded timestamp. */
type ScalarKey = [ScalarMetricName, number]

/** Primary key of the per-sink stores: the sink name plus the rounded timestamp. */
export type SinkKey = [string, number]

/** Primary-key path of the scalar stores; also the range used for per-series pruning. */
export const SCALAR_INDEX = '[metric+timestamp]'

/** Primary-key path of the per-sink stores; also the range used for per-sink pruning. */
export const SINK_INDEX = '[sink+timestamp]'

/** Every bucket window, in the order observations are written to them. */
const BUCKET_SIZES: MetricBucket[] = ['1m', '5m', '10m']

/** Scalar-metric bucket stores, shared by every series in `ScalarMetricName`. */
export const SCALAR_TABLES: Record<MetricBucket, () => Table<ScalarMetricDto, ScalarKey>> = {
    '1m': () => db.metricScalar1m,
    '5m': () => db.metricScalar5m,
    '10m': () => db.metricScalar10m,
}

/** Per-sink delivery bucket stores. */
export const SINK_DELIVERY_TABLES: Record<MetricBucket, () => Table<SinkMetricDto, SinkKey>> = {
    '1m': () => db.metricSinkDelivery1m,
    '5m': () => db.metricSinkDelivery5m,
    '10m': () => db.metricSinkDelivery10m,
}

/** Per-sink latency bucket stores. */
export const SINK_LATENCY_TABLES: Record<MetricBucket, () => Table<SinkMetricDto, SinkKey>> = {
    '1m': () => db.metricSinkLatency1m,
    '5m': () => db.metricSinkLatency5m,
    '10m': () => db.metricSinkLatency10m,
}

/** Pair each store of a family with its window size, ready for writes and pruning. */
const bucketsOf = <T, TKey>(tables: Record<MetricBucket, () => Table<T, TKey>>): Bucket<T, TKey>[] =>
    BUCKET_SIZES.map((size) => ({ table: tables[size], sizeMs: METRIC_BUCKET_MS[size] }))

export const SCALAR_BUCKETS = bucketsOf(SCALAR_TABLES)

export const SINK_DELIVERY_BUCKETS = bucketsOf(SINK_DELIVERY_TABLES)

export const SINK_LATENCY_BUCKETS = bucketsOf(SINK_LATENCY_TABLES)

/** Every scalar bucket store, for a transaction spanning one poll's whole scalar batch. */
export const SCALAR_STORES = () => SCALAR_BUCKETS.map(({ table }) => table())

/** Every per-sink bucket store — delivery and latency are always written in one pass. */
export const SINK_STORES = () => [...SINK_DELIVERY_BUCKETS, ...SINK_LATENCY_BUCKETS].map(({ table }) => table())

/** Round a timestamp down to the start of its bucket window. */
export const floorToBucket = (ts: number, sizeMs: number): number => Math.floor(ts / sizeMs) * sizeMs
