import { create } from 'zustand'
import { persist } from 'zustand/middleware'

import type { MetricsRequestBodyModel } from '@/shared/api'

/** Chart steps the live badge offers; checked against the bucket widths `/api/metrics` accepts. */
export const CHART_STEPS = ['1m', '5m', '10m'] as const satisfies readonly MetricsRequestBodyModel['duration'][]

export type ChartStep = (typeof CHART_STEPS)[number]

/** Step selected until the user picks another one. */
const DEFAULT_CHART_STEP: ChartStep = '1m'

type LiveControlsState = {
    /** Aggregation step used by the live charts. */
    chartStep: ChartStep
    setChartStep: (value: ChartStep) => void
}

/**
 * Live controls state shared by the `live` badge dropdown and the charts that
 * consume the selected aggregation step. Persisted to localStorage so the step
 * survives page reloads.
 */
export const useLiveControlsStore = create<LiveControlsState>()(
    persist(
        (set) => ({
            chartStep: DEFAULT_CHART_STEP,
            setChartStep: (value) => set({ chartStep: value }),
        }),
        {
            name: 'hump.live-controls',
            version: 1,
            migrate: (persisted) => migrateLiveControls(persisted),
            partialize: (state) => ({ chartStep: state.chartStep }),
        },
    ),
)

/** Read-only access to the selected chart step, for charts that only consume it. */
export const useLiveControlsChartStep = (): ChartStep => useLiveControlsStore((state) => state.chartStep)

/** Carries a stored chart step over from an older version, or the default when it is not a known step. */
export const migrateLiveControls = (persisted: unknown): Pick<LiveControlsState, 'chartStep'> => {
    const stored = (persisted as { chartStep?: unknown } | null)?.chartStep
    const chartStep = CHART_STEPS.find((step) => step === stored) ?? DEFAULT_CHART_STEP

    return { chartStep }
}
