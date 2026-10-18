import { experimental_streamedQuery as streamedQuery, useQueries, useQuery } from '@tanstack/react-query'

import { create } from 'zustand'
import { createJSONStorage, persist } from 'zustand/middleware'

import { errorMessage, Snapshot as SnapshotApi, type SnapshotTablesResponseBodyModel } from '@/shared/api'
import { keys } from '@/shared/state'

import type { SnapshotTable, TableEstimate } from './types'

/** Row count of a table under a filter, as the estimate stream reports it. */
type EstimateResult = { count: number; rough: boolean }

type TablesStore = {
    filter: string
    setFilter: (value: string) => void

    selectedIds: Set<string>
    toggle: (id: string) => void
    toggleAll: (ids: string[]) => void

    wheres: Record<string, string>
    setWhere: (id: string, where: string) => void
}

/** Part of the store kept in localStorage; the `Set` is stored as an array. */
type PersistedTables = { selectedIds: string[]; wheres: Record<string, string> }

/**
 * Table selection, per-table row filters and the name filter of the snapshot start view.
 * Selection and row filters survive page reloads; the name filter does not.
 */
export const useTables = create<TablesStore>()(
    persist<TablesStore, [], [], PersistedTables>(
        (set) => ({
            filter: '',
            selectedIds: new Set(),
            wheres: {},

            setFilter: (value) => set({ filter: value }),

            toggle: (id) =>
                set((prev) => {
                    const next = new Set(prev.selectedIds)
                    if (next.has(id)) next.delete(id)
                    else next.add(id)
                    return { selectedIds: next }
                }),

            // Deselects the given tables when all of them are selected, otherwise selects them;
            // tables outside `ids` keep their selection.
            toggleAll: (ids) =>
                set((prev) => {
                    const next = new Set(prev.selectedIds)
                    if (ids.every((id) => prev.selectedIds.has(id))) for (const id of ids) next.delete(id)
                    else for (const id of ids) next.add(id)
                    return { selectedIds: next }
                }),

            setWhere: (id, where) =>
                set((prev) => (prev.wheres[id] === where ? prev : { wheres: { ...prev.wheres, [id]: where } })),
        }),
        {
            name: 'snapshot-tables',
            version: 1,
            storage: createJSONStorage(() => localStorage),
            partialize: ({ selectedIds, wheres }) => ({ selectedIds: [...selectedIds], wheres }),
            merge: (persisted, current) => {
                const { selectedIds = [], wheres = {} } = (persisted ?? {}) as Partial<PersistedTables>
                return { ...current, selectedIds: new Set(selectedIds), wheres }
            },
        },
    ),
)

/** Configured tables with their suggested filters, fetched once per page visit. */
export const useAvailableTables = () =>
    useQuery({
        queryKey: keys.queries.snapshotTables(),
        queryFn: async ({ signal }) => {
            const { data } = await SnapshotApi.tables({ signal, throwOnError: true })
            return data
        },
        select: (data: SnapshotTablesResponseBodyModel) =>
            (data.tables ?? []).map(
                (model): SnapshotTable => ({
                    id: `${model.schema}.${model.table}`,
                    schema: model.schema,
                    table: model.table,
                    name: `${model.schema}.${model.table}`,
                    suggestedWhere: model.where,
                    columns: (model.columns ?? []).map(({ name, type }) => ({
                        name,
                        type,
                    })),
                    sinks: (model.sinks ?? []).map(({ type, host, target }) => ({
                        type,
                        host,
                        target,
                    })),
                }),
            ),
        staleTime: 0,
        refetchOnWindowFocus: false,
    })

/** Selected tables in canonical display order. */
export const useSelectedTables = (): SnapshotTable[] => {
    const { data: tables = [] } = useAvailableTables()
    const selectedIds = useTables((store) => store.selectedIds)

    return tables.filter((table) => selectedIds.has(table.id))
}

/**
 * Row count per selected table under its row filter, keyed by table id. A table is counted
 * as soon as it is selected and again whenever its row filter changes.
 */
export const useTablesEstimate = (): Record<string, TableEstimate> => {
    const selected = useSelectedTables()
    const wheres = useTables((store) => store.wheres)

    return useQueries({
        queries: selected.map((table) => {
            const where = wheres[table.id] ?? table.suggestedWhere
            return {
                queryKey: keys.queries.snapshotEstimate(table.schema, table.table, where),
                queryFn: streamedQuery<EstimateResult, EstimateResult | undefined>({
                    reducer: (_, chunk) => chunk,
                    initialValue: undefined,
                    streamFn: async function* ({ signal }) {
                        const { stream } = await SnapshotApi.estimate({
                            body: {
                                tables: [{ schema: table.schema, table: table.table, where }],
                            },
                            signal,
                            sseMaxRetryAttempts: 1,
                            onSseError: (error) => {
                                if (!signal.aborted) console.warn('[snapshot] estimate stream error', error)
                            },
                        })

                        let isReceived = false
                        for await (const event of stream) {
                            if ('error' in event) throw new Error(event.error)
                            if ('count' in event) {
                                isReceived = true
                                yield { count: Number(event.count), rough: event.rough }
                            }
                        }

                        if (!isReceived) throw new Error('No estimate received')
                    },
                }),
                retry: false,
                refetchOnReconnect: false,
                refetchOnWindowFocus: false,
            }
        }),
        combine: (results) =>
            Object.fromEntries(
                results.map((result, index): [string | undefined, TableEstimate] => {
                    const id = selected[index]?.id
                    if (result.isError) return [id, { status: 'error', error: errorMessage(result.error) }]
                    if (result.isPending || !result.data) return [id, { status: 'loading' }]

                    const { count, rough } = result.data
                    return [id, { status: 'ready', count, rough, fetching: result.isFetching }]
                }),
            ),
    })
}
