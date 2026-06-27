/**
 * Cross-import API (FSD `@x`) exposed to the `stats` entity slices: the chart step
 * they bucket their series by. Nothing else of this slice is shared.
 */
export { type ChartStep, useLiveControlsChartStep } from '../model/store'
