import type { ChartMotionSpringTransition, ChartRenderer } from '@tanstack/charts'
import { motion } from '@tanstack/charts/motion'

import { useMemo } from 'react'

import type { ChartRow } from './chart-series'

/** Spring the marks settle with on mount and on every data update. */
export const chartSpring = {
    type: 'spring',
    stiffness: 190,
    damping: 24,
    mass: 0.82,
    restDelta: 0.02,
    restSpeed: 0.02,
} satisfies ChartMotionSpringTransition

/** Snappier spring the hover cursor follows the pointer with. */
export const chartCursorSpring = {
    type: 'spring',
    stiffness: 260,
    damping: 26,
    mass: 0.72,
    restDelta: 0.02,
    restSpeed: 0.02,
} satisfies ChartMotionSpringTransition

/** Motion renderer that grows the marks in on mount and springs them between data updates. */
export const useChartRenderer = (): ChartRenderer<ChartRow, number, number> =>
    useMemo(
        () =>
            motion<ChartRow, number, number>({
                initial: 'always',
                transition: chartSpring,
                respectReducedMotion: true,
            }),
        [],
    )
