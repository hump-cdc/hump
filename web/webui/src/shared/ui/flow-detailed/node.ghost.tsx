import type { FC } from 'react'

import { Handle, type Node, type NodeProps, Position } from '@xyflow/react'

import { type FlowNodeBase, type FlowNodeConfig, SinkIcon } from '@/shared/ui/flow'

export type GhostNodeData = {
    /** Sink type shown as a placeholder (e.g. `clickhouse`, `s3`). */
    type: string
}

type GhostNodeModel = Node<GhostNodeData, 'ghost'>

/** Dashed, dimmed placeholder for a sink that isn't configured yet; tables can still connect to it. */
export const GhostNodeView: FC<NodeProps<GhostNodeModel>> = ({ data }) => (
    <div className="flex h-13.5 w-95 items-center gap-2.5 rounded-xl border border-hairline-strong border-dashed bg-surface-1 px-4 opacity-50">
        <span className="flex text-ink-subtle [&_svg]:size-4">
            <SinkIcon type={data.type} />
        </span>
        <span className="font-mono text-ink-subtle text-sm">{data.type}</span>

        <Handle type="target" position={Position.Left} />
    </div>
)

/** Declarative config element: `<FlowDetailedGhost id type />`. */
export const GhostNode: FlowNodeConfig<'ghost', GhostNodeData> = Object.assign(
    (_props: FlowNodeBase & GhostNodeData) => null,
    { flowNodeType: 'ghost' as const },
)
