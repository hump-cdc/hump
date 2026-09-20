/**
 * Cross-import API (FSD `@x`) exposed to the `list` slice: the filter store its query
 * reads and the option lists that tell a full facet from a narrowed one.
 */
export { operationItems, statusItems } from '../model/options'
export { type OperationsFilterState, useOperationsFilter } from '../model/store'
