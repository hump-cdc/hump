import type { SinkNodeData, SinkTarget } from './node.sink'

export type { ClearSelectionHandler, EdgeSelectHandler, TableNodeSelectHandler } from '@/shared/ui/flow'

/** Fired when a sink node is selected; carries its id and rendered data. */
export type SinkNodeSelectHandler = (id: string, data: SinkNodeData) => void
/**
 * Fired when a target row inside a sink is selected; carries the sink id and the target.
 * Its selection id is `sinkTargetKey(sinkId, target.id)`.
 */
export type SinkTargetSelectHandler = (sinkId: string, target: SinkTarget) => void

/** Imperative API exposed via `ref`. */
export type FlowDetailedRef = {
    /**
     * Animate the edge from `tableId` (source) to `sinkId`'s `targetId` (target handle) as a
     * dashed marching-ants line for `durationMs`, then revert. Omit `targetId` for edges into a
     * ghost. No-op if no such edge exists.
     */
    animate: (tableId: string, sinkId: string, targetId?: string, durationMs?: number) => void
}
