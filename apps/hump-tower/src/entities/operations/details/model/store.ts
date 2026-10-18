import { create } from 'zustand'

import type { Operation } from '@/entities/operations/list/@x/details'

type OperationDetailsState = {
    /** Operation selected in the list, or `null` when nothing is selected. */
    selected: Operation | null
    select: (operation: Operation) => void
    clear: () => void
}

/**
 * Selected operation: the list writes the clicked row here and highlights it. The row is kept
 * whole, since a finished delivery never changes and there is no endpoint to fetch one by id.
 */
export const useOperationDetailsStore = create<OperationDetailsState>((set) => ({
    selected: null,
    select: (operation) => set({ selected: operation }),
    clear: () => set({ selected: null }),
}))
