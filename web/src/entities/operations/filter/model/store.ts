import { create } from 'zustand'

import { operationItems, statusItems } from './options'

/**
 * Operations filter state, consumed by the operations list to filter the
 * IndexedDB feed. `sinks`/`tables` start empty and are hydrated once from the
 * loaded config (see `hydrateSinks`/`hydrateTables`); `operations`/`statuses`
 * default to every known value. An empty facet matches nothing.
 */
export type OperationsFilterState = {
    /** Selected sink types. */
    sinks: string[]
    /** Selected tables, formatted as `schema.table`. */
    tables: string[]
    /** Selected operation kinds (insert/update/delete/truncate). */
    operations: string[]
    /** Selected delivery statuses (delivered/failed). */
    statuses: string[]
    /** Free-text search; space-separated words are ANDed (substring, case-insensitive). */
    search: string
    /** Whether `sinks` has been seeded from config yet. */
    sinksHydrated: boolean
    /** Whether `tables` has been seeded from config yet. */
    tablesHydrated: boolean
    setSinks: (value: string[]) => void
    setTables: (value: string[]) => void
    setOperations: (value: string[]) => void
    setStatuses: (value: string[]) => void
    setSearch: (value: string) => void
    /** Seed `sinks` with all available sinks once; no-op after the first non-empty seed. */
    hydrateSinks: (all: string[]) => void
    /** Seed `tables` with all available tables once; no-op after the first non-empty seed. */
    hydrateTables: (all: string[]) => void
}

export const useOperationsFilter = create<OperationsFilterState>((set, get) => ({
    sinks: [],
    tables: [],
    operations: operationItems.map((item) => item.value),
    statuses: statusItems.map((item) => item.value),
    search: '',
    sinksHydrated: false,
    tablesHydrated: false,

    setSinks: (value) => set({ sinks: value }),
    setTables: (value) => set({ tables: value }),
    setOperations: (value) => set({ operations: value }),
    setStatuses: (value) => set({ statuses: value }),
    setSearch: (value) => set({ search: value }),

    hydrateSinks: (all) => {
        if (get().sinksHydrated || all.length === 0) return
        set({ sinks: all, sinksHydrated: true })
    },
    hydrateTables: (all) => {
        if (get().tablesHydrated || all.length === 0) return
        set({ tables: all, tablesHydrated: true })
    },
}))
