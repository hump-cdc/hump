import { create } from 'zustand'

import type { SinkNodeData, TableNodeData } from '@/shared/ui/flow'

import type { SelectedItem } from './types'

type FlowInspectorState = {
    /** Currently inspected node/edge, or `null` when nothing is selected. */
    selected: SelectedItem | null
    selectTable: (id: string, data: TableNodeData) => void
    selectSink: (id: string, data: SinkNodeData) => void
    selectEdge: (item: { id: string; sourceLabel: string; targetLabel: string }) => void
    clear: () => void
}

/**
 * Shared selection state for the flows topology: the canvas writes the clicked
 * node/edge here, and the inspector drawer reads it. The drawer stays mounted only
 * while `selected` is non-null.
 */
export const useFlowInspectorStore = create<FlowInspectorState>((set) => ({
    selected: null,
    selectTable: (id, data) => set({ selected: { kind: 'table', id, data } }),
    selectSink: (id, data) => set({ selected: { kind: 'sink', id, data } }),
    selectEdge: (item) => set({ selected: { kind: 'edge', ...item } }),
    clear: () => set({ selected: null }),
}))
