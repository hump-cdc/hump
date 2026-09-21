import type { TableNodeData } from './node.table'

/** A declarative node-config child resolved to its React Flow node type and data (no position yet). */
export type FlowNodeSpec = { id: string; type: string; data: Record<string, unknown> }

/** Fired when a table node is selected; carries its id and rendered data. */
export type TableNodeSelectHandler = (id: string, data: TableNodeData) => void
/** Fired when an edge is selected; carries its id, connected node ids and the target handle id, if any. */
export type EdgeSelectHandler = (
    id: string,
    edge: { source: string; target: string; targetHandle?: string | null },
) => void
/** Fired when the current selection is cleared (re-click selected item or click empty pane). */
export type ClearSelectionHandler = () => void
