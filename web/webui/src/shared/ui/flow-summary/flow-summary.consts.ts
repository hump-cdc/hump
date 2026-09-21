import type { NodeTypes } from '@xyflow/react'

import { LabelNodeView, TableNodeView } from '@/shared/ui/flow'

import { SinkNodeView } from './node.sink'

export const nodeTypes: NodeTypes = {
    table: TableNodeView,
    sink: SinkNodeView,
    label: LabelNodeView,
}

/** Auto-layout geometry: vertical step between consecutive nodes of each column. */
export const TABLE_ROW_STEP = 136
export const SINK_ROW_STEP = 190
