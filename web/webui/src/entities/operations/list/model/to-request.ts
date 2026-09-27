import type { OperationsFilterState } from '@/entities/operations/filter/@x/list'
import { operationItems, statusItems } from '@/entities/operations/filter/@x/list'
import type { OperationsFilterKey } from '@/shared/state'

/** Filter store fields the request is built from. */
export type OperationsFilterInput = Pick<
    OperationsFilterState,
    'sinks' | 'tables' | 'operations' | 'statuses' | 'sinksHydrated' | 'tablesHydrated'
>

/** API status of each filter status value. */
const API_STATUSES: Record<string, 'success' | 'failure'> = {
    delivered: 'success',
    failed: 'failure',
}

/**
 * Request filter for the store's state, or `undefined` when an empty facet
 * matches nothing. A facet that allows everything is left out, as is a sink or
 * table facet not yet seeded from config; lists are sorted so query keys stay stable.
 */
export const toOperationsFilter = (state: OperationsFilterInput, text: string): OperationsFilterKey | undefined => {
    const isSinkFacetEmpty = state.sinksHydrated && state.sinks.length === 0
    const isTableFacetEmpty = state.tablesHydrated && state.tables.length === 0
    if (isSinkFacetEmpty || isTableFacetEmpty || state.operations.length === 0 || state.statuses.length === 0) {
        return undefined
    }

    const filter: OperationsFilterKey = {}
    if (state.sinksHydrated) {
        filter.sinks = sorted(state.sinks)
    }
    if (state.tablesHydrated) {
        filter.tables = sorted(state.tables)
    }
    if (!coversAll(state.operations, operationItems)) {
        filter.operations = sorted(state.operations) as NonNullable<OperationsFilterKey['operations']>
    }
    if (!coversAll(state.statuses, statusItems)) {
        filter.statuses = sorted(state.statuses.flatMap((status) => API_STATUSES[status] ?? []))
    }
    const trimmed = text.trim()
    if (trimmed) {
        filter.text = trimmed
    }

    return filter
}

/** Whether `selected` includes every option. */
const coversAll = (selected: readonly string[], options: readonly { value: string }[]): boolean =>
    options.every((option) => selected.includes(option.value))

/** Sorted copy of `values`. */
const sorted = <V extends string>(values: readonly V[]): V[] => [...values].sort()
