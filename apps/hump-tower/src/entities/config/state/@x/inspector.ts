/**
 * Cross-import API (FSD `@x`) exposed to the flows inspector: the chart step, which keys the
 * metrics query the inspector shares with the canvas. Nothing else of this slice is shared.
 */
export { useLiveControlsChartStep } from '../model/store'
