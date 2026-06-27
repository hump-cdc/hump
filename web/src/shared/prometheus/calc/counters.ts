import type { MetricFamily } from '../metrics'
import { seriesOf } from '../query/samples'
import type { CounterSnapshot, PromMetrics, PromSeries } from '../types'

/**
 * Every cumulative reading one scrape carries for `families` — the baseline the next poll
 * measures against. Gauge families are left out by the caller: they are stored as read and
 * need no baseline.
 */
export const readingsOf = (metrics: PromMetrics, families: readonly MetricFamily[]): CounterSnapshot =>
    Object.fromEntries(families.flatMap((name) => seriesOf(metrics, name)).map(({ key, value }) => [key, value]))

/**
 * How much every cumulative counter advanced since the previous poll. Built once per scrape
 * so the baseline and the restart verdict are stated in one place rather than threaded
 * through every caller that needs a delta.
 */
export type CounterWindow = {
    /** Sum of how far a family of counters advanced. */
    delta: (series: PromSeries[]) => number
    /**
     * {@link delta} with the advances kept apart per series instead of summed. A series with
     * no baseline is absent rather than `0`, separating "did not advance" from "not measured".
     */
    perSeries: (series: PromSeries[]) => Map<string, number>
}

/**
 * Measure this poll against `previous`. Set `restarted` when the exporter's process identity
 * changed: every baseline then drops to zero, because comparing readings alone misses a
 * restart that climbs past the old value inside one poll window and would undercount it.
 */
export const counterWindow = (previous: CounterSnapshot, restarted: boolean): CounterWindow => {
    // The baseline a series is measured against: zero after a restart, otherwise its previous
    // reading. A series scraped for the first time has none.
    const baselineOf = (key: string): number | undefined => (restarted ? 0 : previous[key])

    // How far one series advanced past its baseline, or `0` when it has no baseline yet.
    const advanceOf = (value: number, baseline: number | undefined): number => {
        if (baseline === undefined) return 0

        // Below its baseline without a declared restart: take the whole reading, the safer guess.
        return value >= baseline ? value - baseline : value
    }

    return {
        delta: (series) => series.reduce((total, { key, value }) => total + advanceOf(value, baselineOf(key)), 0),

        perSeries: (series) => {
            const deltas = new Map<string, number>()

            for (const { key, value } of series) {
                const baseline = baselineOf(key)
                if (baseline !== undefined) deltas.set(key, advanceOf(value, baseline))
            }

            return deltas
        },
    }
}
