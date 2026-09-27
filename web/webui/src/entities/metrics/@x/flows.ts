/**
 * Cross-import API (FSD `@x`) exposed to the `flows` entity slices: the metrics query and the
 * pipeline topology it carries. Nothing else of this slice is shared.
 */
export type { MetricsSink, MetricsTopology } from '../model/types'
export { useMetrics } from '../model/use-metrics'
