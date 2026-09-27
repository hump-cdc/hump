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
            partialize: (state) => ({ tables: state.tables, sinks: state.sinks }),
        },
    ),
)

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
