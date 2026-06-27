import {
    counterWindow,
    FAMILY,
    gauge,
    type PromMetrics,
    STATUS_FAILURE,
    STATUS_SUCCESS,
    seriesOf,
    sumOf,
} from '@/shared/prometheus'
import {
    db,
    FLOW_STORES,
    getCounterSnapshot,
    SCALAR_STORES,
    type ScalarMetricName,
    SINK_STORES,
    saveScalar,
    saveSinkDelivery,
    saveSinkLatency,
    saveSinkStats,
    saveTableCounts,
    saveTargetStats,
} from '@/shared/storage'

import { flowUpdate as calcFlowUpdate } from './persist.flows'
import { prime } from './prime'

/** Store a gauge reading, skipping a family the scrape did not carry. */
const saveGauge = (metric: ScalarMetricName, value: number | undefined, now: number): Promise<void> =>
    value === undefined ? Promise.resolve() : saveScalar(metric, value, now)

/**
 * Persist everything one scrape yields: the scalar series and the flow graph's node and edge
 * stats. Gauges are stored as read, counters as their advance since the stored snapshot.
 */
export const persist = async (metrics: PromMetrics, now: number): Promise<void> => {
    const snapshot = await getCounterSnapshot()

    // A change in the exporter's start time is the only reliable sign that the process
    // restarted and its counters went back to zero. Claimed only when both scrapes identified
    // their process and the two differ.
    const startedAt = gauge(metrics, FAMILY.processStart)
    const restarted = snapshot.startedAt !== undefined && startedAt !== undefined && snapshot.startedAt !== startedAt

    const window = counterWindow(snapshot.counters, restarted)

    const rows = window.delta(seriesOf(metrics, FAMILY.rows))
    const delivered = window.delta(seriesOf(metrics, FAMILY.sinkMessages, (labels) => labels.status === STATUS_SUCCESS))
    const failed = window.delta(seriesOf(metrics, FAMILY.sinkMessages, (labels) => labels.status === STATUS_FAILURE))
    const flows = calcFlowUpdate(metrics, window)

    // One transaction per store group collapses ~50 commits per poll into four, and each
    // commit re-runs the dashboard's matching `useLiveQuery` calls — the real cost.
    //
    // The groups stay sequential and separately awaited: only a root transaction resolves on
    // its real commit, so running them together loses the ordering `prime` below relies on.
    await db.transaction('rw', SCALAR_STORES(), async () => {
        await saveGauge('commitLag', gauge(metrics, FAMILY.commitLag), now)
        await saveGauge('senderLag', gauge(metrics, FAMILY.senderLag), now)
        await saveGauge('heap', gauge(metrics, FAMILY.heap), now)
        await saveScalar('asyncQueue', sumOf(metrics, FAMILY.asyncQueue), now)

        await saveScalar('rows', rows, now, 'sum')
        await saveScalar('operationsDelivered', delivered, now, 'sum')
        await saveScalar('operationsFailed', failed, now, 'sum')
        // Share of failed deliveries this poll, 0–100; a poll that delivered nothing records
        // 0, keeping the series continuous.
        await saveScalar('failRate', delivered <= 0 ? 0 : (failed / delivered) * 100, now)
    })

    // The scrape covers everything touched since the pipeline started, so this corrects what
    // the operations stream counted and fills in whatever it never saw.
    await db.transaction('rw', FLOW_STORES(), async () => {
        await saveTableCounts(flows.tables)
        await saveSinkStats(flows.sinks)
        await saveTargetStats(flows.targets)
    })

    // Per-sink-type series. A zero delivery count keeps an idle sink continuous, but a window
    // with no observed durations leaves latency untouched rather than recording a fake zero.
    await db.transaction('rw', SINK_STORES(), async () => {
        for (const { sinkType, delivered, p95 } of flows.types) {
            await saveSinkDelivery(sinkType, delivered, now)
            if (p95 !== undefined) await saveSinkLatency(sinkType, p95, now)
        }
    })

    // Last, so a crash before this point recomputes the window rather than losing it.
    await prime(metrics)
}
