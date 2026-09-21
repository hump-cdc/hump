import { useMemo } from 'react'

import { useLiveControlsChartStep } from '@/entities/config/state/@x/flows'
import { useMetrics } from '@/entities/metrics/@x/flows'

import { type FlowsTopology, toFlowsTopology } from './topology'

/** Rendered while the metrics are loading or unavailable. */
const EMPTY: FlowsTopology = { tables: [], sinks: [], edges: [], detailedSinks: [], detailedEdges: [] }

/**
 * Flows topology from the metrics endpoint, at the chart step from the live controls (which sets
 * the window the p95 latencies cover). Empty until the first response arrives.
 */
export const useFlowsTopology = (): FlowsTopology => {
    const chartStep = useLiveControlsChartStep()
    const { data } = useMetrics(chartStep)
    const topology = data?.topology

    return useMemo(() => (topology ? toFlowsTopology(topology) : EMPTY), [topology])
}
