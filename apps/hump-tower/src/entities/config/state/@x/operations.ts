/**
 * Cross-import API (FSD `@x`) exposed to the `operations` entity slices: the chart step, so the
 * target options read the metrics query the rest of the UI already polls. Nothing else of this slice is shared.
 */
export { useLiveControlsChartStep } from '../model/store'
