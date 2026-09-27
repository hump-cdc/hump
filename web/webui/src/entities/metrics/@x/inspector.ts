/**
 * Cross-import API (FSD `@x`) exposed to the flows inspector: the metrics query and the topology
 * types it resolves the selected node or edge against. Nothing else of this slice is shared.
 */
export type { MetricsConnection, MetricsDelivery, MetricsTable, MetricsTopology } from '../model/types'
export { useMetrics } from '../model/use-metrics'
