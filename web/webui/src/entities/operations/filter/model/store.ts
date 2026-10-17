import { create } from 'zustand'
import { persist } from 'zustand/middleware'

import { operationItems, statusItems } from './options'

/** localStorage key holding the persisted filter values. */
const STORAGE_KEY = 'operations:filter'

/**
 * Filter values of the operations log. `sinks`/`targets`/`tables` are `null` while they allow
 * every option, since the options only arrive with the config and metrics topology;
 * `operations`/`statuses` default to every known value. An empty facet matches nothing.
 */
export type OperationsFilterValues = {
    /** Selected sink types; `null` allows all of them. */
    sinks: string[] | null
    /** Selected sink targets, as the metrics topology reports them; `null` allows all of them. */
    targets: string[] | null
    /** Selected tables, formatted as `schema.table`; `null` allows all of them. */
    tables: string[] | null
    /** Selected operation kinds (insert/update/delete/truncate). */
    operations: string[]
    /** Selected delivery statuses (delivered/failed). */
    statuses: string[]
    /** Free-text search: a case-insensitive substring of the payload, target, error or metadata. */
    search: string
}

/** Filter that allows every option: every facet at "All" and no search text. */
export const defaultFilterValues: OperationsFilterValues = {
    sinks: null,
    targets: null,
    tables: null,
    operations: operationItems.map((item) => item.value),
    statuses: statusItems.map((item) => item.value),
    search: '',
}

/**
 * Operations filter state, consumed by the operations list to filter the `/api/operations` log.
 * The values persist to localStorage, so the filter survives reloads.
 */
export type OperationsFilterState = OperationsFilterValues & {
    setSinks: (value: string[] | null) => void
    setTargets: (value: string[] | null) => void
    setTables: (value: string[] | null) => void
    setOperations: (value: string[]) => void
    setStatuses: (value: string[]) => void
    setSearch: (value: string) => void
    /** Overwrites the given facets at once, keeping the others. */
    apply: (patch: Partial<OperationsFilterValues>) => void
}

export const useOperationsFilter = create<OperationsFilterState>()(
    persist(
        (set) => ({
            ...defaultFilterValues,

            setSinks: (value) => set({ sinks: value }),
            setTargets: (value) => set({ targets: value }),
            setTables: (value) => set({ tables: value }),
            setOperations: (value) => set({ operations: value }),
            setStatuses: (value) => set({ statuses: value }),
            setSearch: (value) => set({ search: value }),
            apply: (patch) => set(patch),
        }),
        {
            name: STORAGE_KEY,
            version: 1,
            partialize: ({ sinks, targets, tables, operations, statuses, search }): OperationsFilterValues => ({
                sinks,
                targets,
                tables,
                operations,
                statuses,
                search,
            }),
        },
    ),
)
