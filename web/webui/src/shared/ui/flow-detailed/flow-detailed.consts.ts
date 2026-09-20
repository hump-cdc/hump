import type { NodeTypes } from '@xyflow/react'

import { GhostNodeView } from './node.ghost'
import { LabelNodeView } from './node.label'
import { SinkNodeView } from './node.sink'
import { TableNodeView } from './node.table'

export const nodeTypes: NodeTypes = {
    table: TableNodeView,
    sink: SinkNodeView,
    ghost: GhostNodeView,
    label: LabelNodeView,
}

/** Auto-layout geometry: table nodes form the left column; sinks, then ghosts, stack in the right column. */
export const TABLE_COLUMN_X = 0
export const SINK_COLUMN_X = 584
export const TABLE_HEIGHT = 104
export const TABLE_GAP = 32
/**
 * Sink height = chrome (border, padding, fixed-height header, divider, section gaps) + one
 * fixed-height row per target, rows separated by `TARGET_GAP`. Must match `node.sink.tsx`.
 */
export const SINK_CHROME_HEIGHT = 81
export const TARGET_ROW_HEIGHT = 28
export const TARGET_GAP = 8
export const SINK_GAP = 8
export const GHOST_HEIGHT = 54
export const GHOST_GAP = 22
/** Vertical offset of a column caption above the column's first node. */
export const LABEL_OFFSET_Y = 36

/**
 * Edge z-index: above every node, including a selected node (React Flow elevates it to
 * 1000), so edges stay visible over the sink card all the way to the target row's handle.
 */
export const EDGE_Z_INDEX = 1001

/** Default duration (ms) an edge stays animated after `animate()` before reverting. */
export const DEFAULT_ANIMATE_DURATION = 1300
