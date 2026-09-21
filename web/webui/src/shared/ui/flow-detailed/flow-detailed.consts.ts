import type { NodeTypes } from '@xyflow/react'

import { LabelNodeView, TableNodeView } from '@/shared/ui/flow'

import { GhostNodeView } from './node.ghost'
import { SinkNodeView } from './node.sink'

export const nodeTypes: NodeTypes = {
    table: TableNodeView,
    sink: SinkNodeView,
    ghost: GhostNodeView,
    label: LabelNodeView,
}

/** Auto-layout geometry: sinks, then ghosts, stack below each other in the right column. */
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

/**
 * Edge z-index: above every node, including a selected node (React Flow elevates it to
 * 1000), so edges stay visible over the sink card all the way to the target row's handle.
 */
export const EDGE_Z_INDEX = 1001
