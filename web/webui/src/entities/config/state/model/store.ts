import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export const CHART_STEPS = ['1m', '5m', '10m'] as const

export type ChartStep = (typeof CHART_STEPS)[number]

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
            chartStep: '1m',
            setChartStep: (value) => set({ chartStep: value }),
        }),
        {
            name: 'hump.live-controls',
            partialize: (state) => ({ chartStep: state.chartStep }),
        },
    ),
)

/** Read-only access to the selected chart step, for charts that only consume it. */
export const useLiveControlsChartStep = (): ChartStep => useLiveControlsStore((state) => state.chartStep)
