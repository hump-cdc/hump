import { COUNTER_FAMILIES, FAMILY, gauge, type PromMetrics, readingsOf } from '@/shared/prometheus'
import { saveCounterSnapshot } from '@/shared/storage'

/**
 * Make this scrape the baseline the next poll measures against. Used alone for a loop's
 * first scrape, whose counters cover the whole uptime and would spike the series.
 */
export const prime = async (metrics: PromMetrics): Promise<void> => {
    await saveCounterSnapshot({
        startedAt: gauge(metrics, FAMILY.processStart),
        counters: readingsOf(metrics, COUNTER_FAMILIES),
    })
}
