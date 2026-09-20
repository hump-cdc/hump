import { type ComponentProps, type FC, type Ref, useMemo } from 'react'

import { Background, BackgroundVariant, Controls, ReactFlow, ReactFlowProvider } from '@xyflow/react'

import { cn } from '@/shared/utils'

import { nodeTypes } from './flow-detailed.consts'
import { TargetSelectionContext } from './flow-detailed.context'
import { partitionChildren, useEdgeAnimation, useFlowSelection, useSelectionHandlers } from './flow-detailed.helpers'
import type {
    ClearSelectionHandler,
    EdgeSelectHandler,
    FlowDetailedRef,
    SinkNodeSelectHandler,
    SinkTargetSelectHandler,
    TableNodeSelectHandler,
} from './flow-detailed.types'

export { sinkTargetKey } from './flow-detailed.context'
export type {
    ClearSelectionHandler,
    EdgeSelectHandler,
    FlowDetailedRef,
    SinkNodeSelectHandler,
    SinkTargetSelectHandler,
    TableNodeSelectHandler,
} from './flow-detailed.types'
export { GhostNode as FlowDetailedGhost, type GhostNodeData } from './node.ghost'
export { SinkNode as FlowDetailedSink, type SinkNodeData, type SinkTarget } from './node.sink'
export { TableNode as FlowDetailedTable, type TableNodeData } from './node.table'

import '@xyflow/react/dist/style.css'
import './flow-detailed.css'

type FlowDetailedProps = Omit<ComponentProps<typeof ReactFlow>, 'nodes' | 'selected' | 'ref'> & {
    /** Id of the currently selected node, edge or target (`sinkTargetKey(sinkId, targetId)`). */
    selected?: string
    onTableNodeSelect?: TableNodeSelectHandler
    onSinkNodeSelect?: SinkNodeSelectHandler
    /** Fired when a target row is selected; the sink card itself stays unselected. */
    onSinkTargetSelect?: SinkTargetSelectHandler
    onEdgeSelect?: EdgeSelectHandler
    onClearSelection?: ClearSelectionHandler
    ref?: Ref<FlowDetailedRef>
}

/**
 * Detailed topology: source tables (left, with insert/update/delete/truncate counts)
 * wired to individual targets (topics, queues, …) inside each sink (right), followed by
 * dashed ghost placeholders for sinks not configured yet. Edges address a target via
 * `targetHandle`; edge weight comes from each edge's `style` (`opacity`/`strokeWidth`);
 * `ref.animate` flashes a route.
 */
export const FlowDetailed: FC<FlowDetailedProps> = ({
    className,
    children,
    selected,
    edges,
    onTableNodeSelect,
    onSinkNodeSelect,
    onSinkTargetSelect,
    onEdgeSelect,
    onClearSelection,
    ref,
    ...props
}) => {
    const { nodes, rest } = useMemo(() => partitionChildren(children), [children])

    const animatedIds = useEdgeAnimation(edges, ref)

    const { selectedNodes, selectedEdges } = useFlowSelection(nodes, edges, selected, animatedIds)

    const { handleNodeClick, handleTargetClick, handleEdgeClick, handlePaneClick } = useSelectionHandlers({
        selected,
        onTableNodeSelect,
        onSinkNodeSelect,
        onSinkTargetSelect,
        onEdgeSelect,
        onClearSelection,
    })

    const targetSelection = useMemo(
        () => ({ selected, onTargetClick: handleTargetClick }),
        [selected, handleTargetClick],
    )

    return (
        <ReactFlowProvider>
            <TargetSelectionContext value={targetSelection}>
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
            </TargetSelectionContext>
        </ReactFlowProvider>
    )
}
