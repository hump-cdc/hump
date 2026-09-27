import { create } from 'zustand'

import { operationItems, statusItems } from './options'

/**
 * Operations filter state, consumed by the operations list to filter the
 * `/api/operations` log. `sinks`/`targets`/`tables` are `null` while they allow every
 * option, since the options only arrive with the config and metrics topology;
 * `operations`/`statuses` default to every known value. An empty facet matches nothing.
 */
export type OperationsFilterState = {
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
    setSinks: (value: string[] | null) => void
    setTargets: (value: string[] | null) => void
    setTables: (value: string[] | null) => void
    setOperations: (value: string[]) => void
    setStatuses: (value: string[]) => void
    setSearch: (value: string) => void
}

export const useOperationsFilter = create<OperationsFilterState>((set) => ({
    sinks: null,
    targets: null,
    tables: null,
    operations: operationItems.map((item) => item.value),
    statuses: statusItems.map((item) => item.value),
    search: '',

    setSinks: (value) => set({ sinks: value }),
    setTargets: (value) => set({ targets: value }),
    setTables: (value) => set({ tables: value }),
    setOperations: (value) => set({ operations: value }),
    setStatuses: (value) => set({ statuses: value }),
    setSearch: (value) => set({ search: value }),
}))
