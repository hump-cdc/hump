import type { FC } from 'react'

import type { Node, NodeProps } from '@xyflow/react'

export type LabelNodeData = {
    text: string
}

type LabelNodeModel = Node<LabelNodeData, 'label'>

/** Non-interactive column caption (e.g. `TABLES`, `SINKS`) placed above a node column. */
export const LabelNodeView: FC<NodeProps<LabelNodeModel>> = ({ data }) => (
    <span className="pointer-events-none font-medium text-ink-tertiary text-xs tracking-caps">{data.text}</span>
)
