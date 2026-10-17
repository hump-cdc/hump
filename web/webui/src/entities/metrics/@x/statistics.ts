/**
 * Cross-import API (FSD `@x`) exposed to the `statistics` entity slices: the metrics query and
 * the point type its charts plot. Nothing else of this slice is shared.
 */
export type { MetricsPoint } from '../model/types'
export { useMetrics } from '../model/use-metrics'
