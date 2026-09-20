import type { SinkNodeData } from './node.sink'
import type { TableNodeData } from './node.table'

/** Fired when a table node is selected; carries its id and rendered data. */
export type TableNodeSelectHandler = (id: string, data: TableNodeData) => void
/** Fired when a sink node is selected; carries its id and rendered data. */
export type SinkNodeSelectHandler = (id: string, data: SinkNodeData) => void
/** Fired when an edge is selected; carries its id and connected node ids. */
export type EdgeSelectHandler = (id: string, edge: { source: string; target: string }) => void
/** Fired when the current selection is cleared (re-click selected item or click empty pane). */
export type ClearSelectionHandler = () => void

/** Imperative API exposed via `ref`. */
export type FlowSummaryRef = {
    /**
     * Animate the edge from `tableId` (source) to `sinkId` (target) as a dashed
     * marching-ants line for `durationMs`, then revert. No-op if no such edge exists.
     */
    animate: (tableId: string, sinkId: string, durationMs?: number) => void
}
