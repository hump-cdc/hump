/**
 * Cross-import API (FSD `@x`) exposed to the `list` slice: the filter store its query
 * reads. Nothing else of this slice is shared.
 */
export { useOperationsFilter } from '../model/store'
