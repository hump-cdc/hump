import {
    type CounterWindow,
    FAMILY,
    type HistogramBucket,
    keyOf,
    mergeBuckets,
    type PromMetrics,
    parseBound,
    quantile,
    samplesOf,
    seriesOf,
} from '@/shared/prometheus'
import type { FlowSinkStats, FlowTableCounts, FlowTargetStats } from '@/shared/storage'

/** Quantile the flow view reports as a sink's delivery latency. */
const LATENCY_QUANTILE = 0.95

/** The row counter each WAL operation label feeds. */
const OPERATION_FIELDS: Record<string, keyof Omit<FlowTableCounts, 'schema' | 'table'>> = {
    insert: 'insertCount',
    update: 'updateCount',
    delete: 'deleteCount',
    truncate: 'truncateCount',
}

/** One sink type's window totals across its hosts and targets, for the per-sink series. */
export type SinkTypeStats = {
    sinkType: string
    /** Messages delivered successfully in the window. */
    delivered: number
    /** 95th-percentile delivery duration in ms, if the window observed any. */
    p95: number | undefined
}

/** Everything one scrape contributes to the flow graph. */
export type FlowUpdate = {
    tables: FlowTableCounts[]
    sinks: FlowSinkStats[]
    targets: FlowTargetStats[]
    types: SinkTypeStats[]
}

/** Delivery totals for one sink, or one target within a sink. */
type Delivery = {
    operationCount: number
    delivered: number
    failed: number
    buckets: HistogramBucket[]
}

const emptyDelivery = (): Delivery => ({ operationCount: 0, delivered: 0, failed: 0, buckets: [] })

const deliveryOf = (into: Map<string, Delivery>, key: string): Delivery => {
    const existing = into.get(key)
    if (existing) return existing

    const created = emptyDelivery()
    into.set(key, created)
    return created
}

/**
 * Per-table CDC counters, read off the row histogram's observation counts — the backend
 * counts a row once however many sinks it fans out to. Unattributed samples are dropped.
 */
const tableCounts = (metrics: PromMetrics): FlowTableCounts[] => {
    const rows = new Map<string, FlowTableCounts>()

    for (const { labels, value } of samplesOf(metrics, FAMILY.rows)) {
        const field = OPERATION_FIELDS[labels.operation]
        if (!field || !labels.schema || !labels.table) continue

        const key = `${labels.schema} ${labels.table}`
        const row = rows.get(key) ?? {
            schema: labels.schema,
            table: labels.table,
            insertCount: 0,
            updateCount: 0,
            deleteCount: 0,
            truncateCount: 0,
        }
        row[field] = value
        rows.set(key, row)
    }

    return [...rows.values()]
}

/**
 * Share of deliveries that succeeded in the window, or `undefined` when it carried none —
 * an idle sink keeps its last known rate instead of reading as a total failure.
 */
const successRateOf = ({ delivered, failed }: Delivery): number | undefined => {
    const attempts = delivered + failed
    return attempts === 0 ? undefined : (delivered / attempts) * 100
}

/** The window's 95th-percentile delivery latency in ms, if it observed any. */
const p95Of = ({ buckets }: Delivery): number | undefined => {
    const seconds = quantile(buckets, LATENCY_QUANTILE)
    return seconds === undefined ? undefined : seconds * 1000
}

/**
 * Fold one scrape into the flow-graph updates it implies: `operationCount` is absolute,
 * while `successRate` and `p95` cover `window`.
 * The caller must not advance the stored baseline until this update has been applied.
 */
export const flowUpdate = (metrics: PromMetrics, window: CounterWindow): FlowUpdate => {
    const messages = window.perSeries(seriesOf(metrics, FAMILY.sinkMessages))
    const durations = window.perSeries(seriesOf(metrics, FAMILY.sinkDuration))

    const sinks = new Map<string, Delivery>()
    const targets = new Map<string, Delivery>()
    const types = new Map<string, Delivery>()
    const sinkOf = new Map<string, { sinkType: string; host: string }>()
    const targetOf = new Map<string, { sinkType: string; host: string; target: string }>()

    for (const sample of samplesOf(metrics, FAMILY.sinkMessages)) {
        const { labels, value } = sample

        // `retry` counts attempts, not outcomes — folding it in double-counts a message.
        if (labels.status !== 'success' && labels.status !== 'failure') continue

        const sinkKey = `${labels.type} ${labels.host}`
        const targetKey = `${sinkKey} ${labels.target}`
        sinkOf.set(sinkKey, { sinkType: labels.type, host: labels.host })
        targetOf.set(targetKey, { sinkType: labels.type, host: labels.host, target: labels.target })

        // `seriesOf` above already keyed this sample; `keyOf` reads that back.
        const delta = messages.get(keyOf(sample)) ?? 0

        for (const delivery of [
            deliveryOf(sinks, sinkKey),
            deliveryOf(targets, targetKey),
            deliveryOf(types, labels.type),
        ]) {
            delivery.operationCount += value
            if (labels.status === 'success') delivery.delivered += delta
            else delivery.failed += delta
        }
    }

    // Latency buckets are cumulative; their advance is still a valid histogram, because
    // every bound advances monotonically.
    const windowBuckets = new Map<string, HistogramBucket[]>()
    for (const sample of samplesOf(metrics, FAMILY.sinkDuration)) {
        const { labels } = sample
        const targetKey = `${labels.type} ${labels.host} ${labels.target}`
        const delta = durations.get(keyOf(sample)) ?? 0
        const buckets = windowBuckets.get(targetKey) ?? []
        buckets.push({ le: parseBound(labels.le), count: delta })
        windowBuckets.set(targetKey, buckets)
    }

    for (const [targetKey, buckets] of windowBuckets) {
        buckets.sort((a, b) => a.le - b.le)

        const meta = targetOf.get(targetKey)
        if (!meta) continue

        deliveryOf(targets, targetKey).buckets = buckets

        const sink = deliveryOf(sinks, `${meta.sinkType} ${meta.host}`)
        sink.buckets = mergeBuckets(sink.buckets, buckets)

        const type = deliveryOf(types, meta.sinkType)
        type.buckets = mergeBuckets(type.buckets, buckets)
    }

    return {
        tables: tableCounts(metrics),
        sinks: [...sinks].flatMap(([key, delivery]) => {
            const meta = sinkOf.get(key)
            return meta
                ? [
                      {
                          ...meta,
                          operationCount: delivery.operationCount,
                          p95: p95Of(delivery),
                          successRate: successRateOf(delivery),
                      },
                  ]
                : []
        }),
        targets: [...targets].flatMap(([key, delivery]) => {
            const meta = targetOf.get(key)
            return meta
                ? [
                      {
                          ...meta,
                          operationCount: delivery.operationCount,
                          p95: p95Of(delivery),
                          successRate: successRateOf(delivery),
                      },
                  ]
                : []
        }),
        types: [...types].map(([sinkType, delivery]) => ({
            sinkType,
            delivered: delivery.delivered,
            p95: p95Of(delivery),
        })),
    }
}
