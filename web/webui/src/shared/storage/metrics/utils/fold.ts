import type { MetricDto } from '../types'

/**
 * How repeated observations in one window fold together: `max` for gauge readings (peak lag/depth),
 * `sum` for per-poll deltas (total accumulated).
 */
export type Aggregation = 'max' | 'sum'

/** Fold a new observation into the value already stored for its bucket window. */
export const fold = (existing: MetricDto | undefined, value: number, aggregate: Aggregation): number => {
    if (!existing) return value
    return aggregate === 'sum' ? existing.value + value : Math.max(existing.value, value)
}
