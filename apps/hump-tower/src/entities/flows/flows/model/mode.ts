import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export const TOPOLOGY_MODES = ['summary', 'detailed'] as const

/** Topology rendering: one card per sink (`summary`) or sinks broken down into targets (`detailed`). */
export type TopologyMode = (typeof TOPOLOGY_MODES)[number]

/** Mode shown until the user switches. */
const DEFAULT_TOPOLOGY_MODE: TopologyMode = 'detailed'

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
            mode: DEFAULT_TOPOLOGY_MODE,
            setMode: (value) => set({ mode: value }),
        }),
        {
            name: 'hump.flows-mode',
            version: 1,
            migrate: (persisted) => migrateTopologyMode(persisted),
            partialize: (state) => ({ mode: state.mode }),
        },
    ),
)

/** Carries a stored mode over from an older version, or the default when it is not a known mode. */
export const migrateTopologyMode = (persisted: unknown): Pick<TopologyModeState, 'mode'> => {
    const stored = (persisted as { mode?: unknown } | null)?.mode
    const mode = TOPOLOGY_MODES.find((known) => known === stored) ?? DEFAULT_TOPOLOGY_MODE

    return { mode }
}
