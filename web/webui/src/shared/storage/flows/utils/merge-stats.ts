import type { FlowDeliveryStats } from '../types'

/** Overwrite the stored delivery gauges, keeping any the scrape could not measure. */
export const mergeStats = <T extends FlowDeliveryStats>(existing: T, stats: FlowDeliveryStats): T => ({
    ...existing,
    operationCount: stats.operationCount,
    p95: stats.p95 ?? existing.p95,
    successRate: stats.successRate ?? existing.successRate,
})
