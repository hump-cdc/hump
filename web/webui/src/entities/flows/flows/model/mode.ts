import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export const TOPOLOGY_MODES = ['summary', 'detailed'] as const

/** Topology rendering: one card per sink (`summary`) or sinks broken down into targets (`detailed`). */
export type TopologyMode = (typeof TOPOLOGY_MODES)[number]

type TopologyModeState = {
    mode: TopologyMode
    setMode: (value: TopologyMode) => void
}

/**
 * Flows topology view mode, switched from the canvas controls. Persisted to localStorage so the
 * chosen mode survives page reloads.
 */
export const useTopologyModeStore = create<TopologyModeState>()(
    persist(
        (set) => ({
            mode: 'detailed',
            setMode: (value) => set({ mode: value }),
        }),
        {
            name: 'hump.flows-mode',
            partialize: (state) => ({ mode: state.mode }),
        },
    ),
)
