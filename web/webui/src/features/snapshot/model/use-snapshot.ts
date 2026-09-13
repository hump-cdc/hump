import { create } from 'zustand'

import { DEFAULT_SELECTED_IDS, SNAPSHOT_TABLES } from './mock'
import { selectedTables } from './selectors'
import type { SnapshotState, TableProgress, TableStatus } from './types'

const TICK_MS = 250
/** Ticks a single table takes to fill (× TICK_MS ≈ its duration). */
const BASE_DURATION_TICKS = 16
/** Each subsequent table starts this many ticks later, so they stagger. */
const STAGGER_TICKS = 2

const tableById = new Map(SNAPSHOT_TABLES.map((table) => [table.id, table]))

/** Interval handle kept outside the store so it never triggers re-renders. */
let timer: ReturnType<typeof setInterval> | null = null

const stopTimer = (): void => {
    if (timer !== null) {
        clearInterval(timer)
        timer = null
    }
}

type RunPlan = { id: string; startTick: number; durationTicks: number }

/** Compute every selected table's progress at a given tick (pure). */
const progressAt = (plans: RunPlan[], tick: number): Record<string, TableProgress> => {
    const result: Record<string, TableProgress> = {}

    for (const plan of plans) {
        const table = tableById.get(plan.id)
        if (!table) continue

        const local = Math.min(1, Math.max(0, (tick - plan.startTick) / plan.durationTicks))
        const rowsRead = Math.round(table.totalRows * local)
        const status: TableStatus = local <= 0 ? 'waiting' : local >= 1 ? 'done' : 'processing'

        result[plan.id] = { id: plan.id, status, rowsRead, itemsSent: rowsRead * table.sinks.length }
    }

    return result
}

const allDone = (plans: RunPlan[], tick: number): boolean =>
    plans.every((plan) => tick >= plan.startTick + plan.durationTicks)

type SnapshotStore = {
    state: SnapshotState
    selectedIds: Set<string>
    filter: string
    /** Per-table runtime progress; populated while processing and kept after finish. */
    progress: Record<string, TableProgress>
    elapsedSec: number
    toggle: (id: string) => void
    toggleAll: () => void
    setFilter: (value: string) => void
    run: () => void
    stop: () => void
    reset: () => void
}

export const useSnapshot = create<SnapshotStore>((set, get) => ({
    state: 'initial',
    selectedIds: new Set(DEFAULT_SELECTED_IDS),
    filter: '',
    progress: {},
    elapsedSec: 0,

    toggle: (id) => {
        if (get().state !== 'initial') return
        set((prev) => {
            const next = new Set(prev.selectedIds)
            if (next.has(id)) next.delete(id)
            else next.add(id)
            return { selectedIds: next }
        })
    },

    toggleAll: () => {
        if (get().state !== 'initial') return
        set((prev) => ({
            selectedIds:
                prev.selectedIds.size === SNAPSHOT_TABLES.length
                    ? new Set()
                    : new Set(SNAPSHOT_TABLES.map((table) => table.id)),
        }))
    },

    setFilter: (value) => set({ filter: value }),

    run: () => {
        const tables = selectedTables(get().selectedIds)
        if (tables.length === 0) return

        stopTimer()

        const plans: RunPlan[] = tables.map((table, index) => ({
            id: table.id,
            startTick: index * STAGGER_TICKS,
            durationTicks: BASE_DURATION_TICKS + index * 2,
        }))

        set({ state: 'processing', elapsedSec: 0, progress: progressAt(plans, 0) })

        let tick = 0
        timer = setInterval(() => {
            tick += 1
            const progress = progressAt(plans, tick)
            const elapsedSec = (tick * TICK_MS) / 1000

            if (allDone(plans, tick)) {
                stopTimer()
                set({ state: 'finished', progress, elapsedSec })
            } else {
                set({ progress, elapsedSec })
            }
        }, TICK_MS)
    },

    stop: () => {
        stopTimer()
        set({ state: 'initial', progress: {}, elapsedSec: 0 })
    },

    reset: () => {
        stopTimer()
        set({ state: 'initial', progress: {}, elapsedSec: 0 })
    },
}))
