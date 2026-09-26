import { type ComponentProps, type FC, type Ref, useMemo } from 'react'

import { Flow, useFlowSelection, useSelectionHandlers } from '@/shared/ui/flow'

import { EDGE_Z_INDEX, nodeTypes } from './flow-detailed.consts'
import { TargetSelectionContext } from './flow-detailed.context'
import {
    getRevealId,
    partitionChildren,
    useDetailedEdgeAnimation,
    useTargetClickHandler,
} from './flow-detailed.helpers'
import type {
    ClearSelectionHandler,
    EdgeSelectHandler,
    FlowDetailedRef,
    SinkNodeSelectHandler,
    SinkTargetSelectHandler,
    TableNodeSelectHandler,
} from './flow-detailed.types'
import type { SinkNodeData } from './node.sink'

export { TableNode as FlowDetailedTable, type TableNodeData } from '@/shared/ui/flow'

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

type FlowDetailedProps = Omit<ComponentProps<typeof Flow>, 'nodes' | 'selected' | 'ref'> & {
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

    const animatedEdges = useDetailedEdgeAnimation(edges, ref)

    const { selectedNodes, selectedEdges } = useFlowSelection(nodes, edges, selected, animatedEdges, EDGE_Z_INDEX)

    const { handleNodeClick, handleEdgeClick, handlePaneClick } = useSelectionHandlers<SinkNodeData>({
        selected,
        onTableNodeSelect,
        onSinkNodeSelect,
        onEdgeSelect,
        onClearSelection,
    })

    const handleTargetClick = useTargetClickHandler(selected, onSinkTargetSelect, onClearSelection)

    const revealId = useMemo(() => getRevealId(nodes, selected), [nodes, selected])

    const targetSelection = useMemo(
        () => ({ selected, onTargetClick: handleTargetClick }),
        [selected, handleTargetClick],
    )

    return (
        <TargetSelectionContext value={targetSelection}>
            <Flow
                nodes={selectedNodes}
                edges={selectedEdges}
                nodeTypes={nodeTypes}
                onNodeClick={handleNodeClick}
                onEdgeClick={handleEdgeClick}
                onPaneClick={handlePaneClick}
                revealId={revealId}
                revealKey={selected}
                {...props}
            >
                {rest}
            </Flow>
        </TargetSelectionContext>
    )
}
