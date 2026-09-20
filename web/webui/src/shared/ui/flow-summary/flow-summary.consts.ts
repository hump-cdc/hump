import type { NodeTypes } from '@xyflow/react'

import { LabelNodeView } from './node.label'
import { SinkNodeView } from './node.sink'
import { TableNodeView } from './node.table'

export const nodeTypes: NodeTypes = {
    table: TableNodeView,
    sink: SinkNodeView,
    label: LabelNodeView,
}

/** Auto-layout geometry: table nodes form the left column, sink nodes the right column. */
export const TABLE_COLUMN_X = 0
export const SINK_COLUMN_X = 584
export const TABLE_ROW_STEP = 136
export const SINK_ROW_STEP = 190
/** Vertical offset of a column caption above the column's first node. */
export const LABEL_OFFSET_Y = 36

/** Default duration (ms) an edge stays animated after `animate()` before reverting. */
export const DEFAULT_ANIMATE_DURATION = 1300
