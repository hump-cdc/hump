import type { SinkNodeData, SinkTarget } from './node.sink'
import type { TableNodeData } from './node.table'

/** Fired when a table node is selected; carries its id and rendered data. */
export type TableNodeSelectHandler = (id: string, data: TableNodeData) => void
/** Fired when a sink node is selected; carries its id and rendered data. */
export type SinkNodeSelectHandler = (id: string, data: SinkNodeData) => void
/**
 * Fired when a target row inside a sink is selected; carries the sink id and the target.
 * Its selection id is `sinkTargetKey(sinkId, target.id)`.
 */
export type SinkTargetSelectHandler = (sinkId: string, target: SinkTarget) => void
/** Fired when an edge is selected; carries its id, connected node ids and the sink target (handle) id. */
export type EdgeSelectHandler = (
    id: string,
    edge: { source: string; target: string; targetHandle?: string | null },
) => void
/** Fired when the current selection is cleared (re-click selected item or click empty pane). */
export type ClearSelectionHandler = () => void

/** Imperative API exposed via `ref`. */
export type FlowDetailedRef = {
    /**
     * Animate the edge from `tableId` (source) to `sinkId`'s `targetId` (target handle) as a
     * dashed marching-ants line for `durationMs`, then revert. Omit `targetId` for edges into a
     * ghost. No-op if no such edge exists.
     */
    animate: (tableId: string, sinkId: string, targetId?: string, durationMs?: number) => void
}
