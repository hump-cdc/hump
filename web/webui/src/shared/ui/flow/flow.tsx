import type { ComponentProps, FC } from 'react'

import { Background, BackgroundVariant, Controls, ReactFlow, ReactFlowProvider } from '@xyflow/react'

import { cn } from '@/shared/utils'

import { useMeasuredNodes } from './flow.helpers'

export { DEFAULT_ANIMATE_DURATION, LABEL_OFFSET_Y, SINK_COLUMN_X, TABLE_COLUMN_X } from './flow.consts'
export {
    collectNodeSpecs,
    columnLabel,
    useEdgeAnimation,
    useFlowSelection,
    useMeasuredNodes,
    useSelectionHandlers,
} from './flow.helpers'
export type { ClearSelectionHandler, EdgeSelectHandler, FlowNodeSpec, TableNodeSelectHandler } from './flow.types'
export type { FlowNodeBase, FlowNodeConfig } from './node'
export { LabelNodeView } from './node.label'
export { TableNode, type TableNodeData, TableNodeView } from './node.table'
export { SinkIcon, SinkIconTile } from './sink-icon'

import '@xyflow/react/dist/style.css'
import './flow.css'

type FlowProps = ComponentProps<typeof ReactFlow>

/**
 * Read-only React Flow canvas shared by the topology views: fixed (non-draggable,
 * non-connectable) nodes fitted into view over a dotted background with zoom controls.
 * Children render as overlays inside the canvas. Measured node sizes survive rebuilt `nodes`,
 * so edges stay mounted (and keep animating) across data refreshes.
 */
export const Flow: FC<FlowProps> = ({ className, children, nodes, onNodesChange, ...props }) => {
    const { measuredNodes, handleNodesChange } = useMeasuredNodes(nodes, onNodesChange)

    return (
        <ReactFlowProvider>
            <div className={cn('h-full w-full', className)}>
                <ReactFlow
                    nodesDraggable={false}
                    nodesConnectable={false}
                    nodesFocusable={true}
                    edgesFocusable={true}
                    fitView
                    fitViewOptions={{ maxZoom: 1 }}
                    proOptions={{ hideAttribution: true }}
                    {...props}
                    nodes={measuredNodes}
                    onNodesChange={handleNodesChange}
                    colorMode="light"
                >
                    <Background variant={BackgroundVariant.Dots} gap={20} size={1} />
                    <Controls showInteractive={false} />
                    {children}
                </ReactFlow>
            </div>
        </ReactFlowProvider>
    )
}
