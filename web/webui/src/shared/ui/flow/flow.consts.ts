import type { NodeTypes } from '@xyflow/react'

import { SinkNodeView } from './node.sink'
import { TableNodeView } from './node.table'

export const nodeTypes: NodeTypes = {
    table: TableNodeView,
    sink: SinkNodeView,
}

/** Auto-layout geometry: table nodes form the left column, sink nodes the right column. */
export const TABLE_COLUMN_X = 0
export const SINK_COLUMN_X = 560
export const TABLE_ROW_STEP = 156
export const SINK_ROW_STEP = 92

/** Default duration (ms) an edge stays animated after `animate()` before reverting. */
export const DEFAULT_ANIMATE_DURATION = 1300
