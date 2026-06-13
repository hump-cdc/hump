import { type ComponentProps, type FC, type Ref, useMemo } from 'react'

import { Background, BackgroundVariant, Controls, ReactFlow, ReactFlowProvider } from '@xyflow/react'

import { cn } from '@/shared/utils'

import { nodeTypes } from './flow.consts'
import { partitionChildren, useEdgeAnimation, useFlowSelection, useSelectionHandlers } from './flow.helpers'
import type {
    ClearSelectionHandler,
    EdgeSelectHandler,
    FlowRef,
    SinkNodeSelectHandler,
    TableNodeSelectHandler,
} from './flow.types'

export type {
    ClearSelectionHandler,
    EdgeSelectHandler,
    FlowRef,
    SinkNodeSelectHandler,
    TableNodeSelectHandler,
} from './flow.types'

import '@xyflow/react/dist/style.css'
import './flow.css'

type FlowProps = Omit<ComponentProps<typeof ReactFlow>, 'nodes' | 'selected' | 'ref'> & {
    /** Id of the currently selected node or edge. */
    selected?: string
    onTableNodeSelect?: TableNodeSelectHandler
    onSinkNodeSelect?: SinkNodeSelectHandler
    onEdgeSelect?: EdgeSelectHandler
    onClearSelection?: ClearSelectionHandler
    ref?: Ref<FlowRef>
}

export const Flow: FC<FlowProps> = ({
    className,
    children,
    selected,
    edges,
    onTableNodeSelect,
    onSinkNodeSelect,
    onEdgeSelect,
    onClearSelection,
    ref,
    ...props
}) => {
    const { nodes, rest } = useMemo(() => partitionChildren(children), [children])

    const animatedIds = useEdgeAnimation(edges, ref)

    const { selectedNodes, selectedEdges } = useFlowSelection(nodes, edges, selected, animatedIds)

    const { handleNodeClick, handleEdgeClick, handlePaneClick } = useSelectionHandlers({
        selected,
        onTableNodeSelect,
        onSinkNodeSelect,
        onEdgeSelect,
        onClearSelection,
    })

    return (
        <ReactFlowProvider>
            <div className={cn('h-full w-full', className)}>
                <ReactFlow
                    nodes={selectedNodes}
                    edges={selectedEdges}
                    nodeTypes={nodeTypes}
                    nodesDraggable={false}
                    nodesConnectable={false}
                    nodesFocusable={true}
                    edgesFocusable={true}
                    fitView
                    fitViewOptions={{ maxZoom: 1 }}
                    proOptions={{ hideAttribution: true }}
                    onNodeClick={handleNodeClick}
                    onEdgeClick={handleEdgeClick}
                    onPaneClick={handlePaneClick}
                    {...props}
                    colorMode="light"
                >
                    <Background variant={BackgroundVariant.Dots} gap={20} size={1} />
                    <Controls showInteractive={false} />
                    {rest}
                </ReactFlow>
            </div>
        </ReactFlowProvider>
    )
}
