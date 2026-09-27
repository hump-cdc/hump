import { create } from 'zustand'

import type { SelectedItem, SelectionOf } from './types'

type FlowInspectorState = {
    /** Currently inspected node/edge/target, or `null` when nothing is selected. */
    selected: SelectedItem | null
    selectTable: (item: SelectionOf<'table'>) => void
    selectSink: (item: SelectionOf<'sink'>) => void
    selectTarget: (item: SelectionOf<'target'>) => void
    selectEdge: (item: SelectionOf<'edge'>) => void
    clear: () => void
}

/**
 * Shared selection state for the flows topology: the canvas writes the clicked
 * node/edge/target here, and the inspector drawer reads it. The drawer stays mounted
 * only while `selected` is non-null.
 */
export const useFlowInspectorStore = create<FlowInspectorState>((set) => ({
    selected: null,
    selectTable: (item) => set({ selected: { kind: 'table', ...item } }),
    selectSink: (item) => set({ selected: { kind: 'sink', ...item } }),
    selectTarget: (item) => set({ selected: { kind: 'target', ...item } }),
    selectEdge: (item) => set({ selected: { kind: 'edge', ...item } }),
    clear: () => set({ selected: null }),
}))
