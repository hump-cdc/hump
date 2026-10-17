import { create } from 'zustand'
import { persist } from 'zustand/middleware'

import type { FlowsTopology } from './topology'

/** Filter selection: `all` tracks every item, including ones appearing later; otherwise the picked ids. */
export type FlowsFilterSelection = 'all' | string[]

type FlowsFilterState = {
    tables: FlowsFilterSelection
    sinks: FlowsFilterSelection
    setTables: (value: FlowsFilterSelection) => void
    setSinks: (value: FlowsFilterSelection) => void
}

/**
 * Tables and sinks shown on the flows topology, picked from the toolbar. Persisted to localStorage so
 * the selection survives page reloads.
 */
export const useFlowsFilterStore = create<FlowsFilterState>()(
    persist(
        (set) => ({
            tables: 'all',
            sinks: 'all',
            setTables: (value) => set({ tables: value }),
            setSinks: (value) => set({ sinks: value }),
        }),
        {
            name: 'hump.flows-filter',
            version: 1,
            migrate: (persisted) => migrateFlowsFilter(persisted),
            partialize: (state) => ({ tables: state.tables, sinks: state.sinks }),
        },
    ),
)

/** Carries a stored selection over from an older version; a facet that is not a selection becomes `all`. */
export const migrateFlowsFilter = (persisted: unknown): Pick<FlowsFilterState, 'tables' | 'sinks'> => {
    const stored = persisted as { tables?: unknown; sinks?: unknown } | null

    return { tables: toSelection(stored?.tables), sinks: toSelection(stored?.sinks) }
}

/** A stored value as a selection, or `all` when it is not a list of ids. */
const toSelection = (value: unknown): FlowsFilterSelection =>
    Array.isArray(value) && value.every((id) => typeof id === 'string') ? value : 'all'

/** Whether an item passes the selection; `all` passes everything. */
const isSelected = (selection: FlowsFilterSelection, id: string): boolean =>
    selection === 'all' || selection.includes(id)

/** Topology narrowed to the selected tables and sinks, dropping edges whose table or sink is filtered out. */
export const filterFlowsTopology = (
    topology: FlowsTopology,
    tables: FlowsFilterSelection,
    sinks: FlowsFilterSelection,
): FlowsTopology => {
    if (tables === 'all' && sinks === 'all') {
        return topology
    }

    const isEdgeSelected = (edge: { source: string; target: string }) =>
        isSelected(tables, edge.source) && isSelected(sinks, edge.target)

    return {
        tables: topology.tables.filter((table) => isSelected(tables, table.id)),
        sinks: topology.sinks.filter((sink) => isSelected(sinks, sink.id)),
        edges: topology.edges.filter(isEdgeSelected),
        detailedSinks: topology.detailedSinks.filter((sink) => isSelected(sinks, sink.id)),
        detailedEdges: topology.detailedEdges.filter(isEdgeSelected),
    }
}
