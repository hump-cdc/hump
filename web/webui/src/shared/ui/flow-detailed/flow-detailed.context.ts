import { createContext, useContext } from 'react'

import type { SinkTarget } from './node.sink'

/** Selection state the canvas shares with sink nodes so their target rows can be selected on their own. */
type TargetSelection = {
    /** Id of the currently selected node, edge or target (see {@link sinkTargetKey}). */
    selected: string | undefined
    onTargetClick: (sinkId: string, target: SinkTarget) => void
}

export const TargetSelectionContext = createContext<TargetSelection>({
    selected: undefined,
    onTargetClick: () => {},
})

export const useTargetSelection = (): TargetSelection => useContext(TargetSelectionContext)

/** Selection id of a sink target: `<sinkId>:<targetId>` (target ids are unique only within their sink). */
export const sinkTargetKey = (sinkId: string, targetId: string): string => `${sinkId}:${targetId}`
