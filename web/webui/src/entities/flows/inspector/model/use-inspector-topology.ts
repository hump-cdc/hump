import { useLiveControlsChartStep } from '@/entities/config/state/@x/inspector'
import { type MetricsTopology, useMetrics } from '@/entities/metrics/@x/inspector'

/**
 * The metrics topology the canvas renders, read through the same query (same chart step), so it
 * costs no extra request and refreshes with the canvas. `undefined` until the first response.
 */
export const useInspectorTopology = (): MetricsTopology | undefined => {
    const chartStep = useLiveControlsChartStep()
    const { data } = useMetrics(chartStep)

    return data?.topology
}
