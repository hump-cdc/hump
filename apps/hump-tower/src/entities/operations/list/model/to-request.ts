import type { OperationsFilterState } from '@/entities/operations/filter/@x/list'
import { operationItems, statusItems } from '@/entities/operations/filter/@x/list'
import type { OperationsFilterKey } from '@/shared/state'

/** Filter store fields the request is built from. */
export type OperationsFilterInput = Pick<
    OperationsFilterState,
    'sinks' | 'targets' | 'tables' | 'operations' | 'statuses'
>

/** API status of each filter status value. */
const API_STATUSES: Record<string, 'success' | 'failure'> = {
    delivered: 'success',
    failed: 'failure',
}

/**
 * Request filter for the store's state, or `undefined` when an empty facet
 * matches nothing. A facet that allows everything is left out; lists are sorted so query keys stay stable.
 */
export const toOperationsFilter = (state: OperationsFilterInput, text: string): OperationsFilterKey | undefined => {
    const facets = [state.sinks, state.targets, state.tables, state.operations, state.statuses]
    if (facets.some((facet) => facet?.length === 0)) {
        return undefined
    }

    const filter: OperationsFilterKey = {}
    if (state.sinks) {
        filter.sinks = sorted(state.sinks)
    }
    if (state.targets) {
        filter.targets = sorted(state.targets)
    }
    if (state.tables) {
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
