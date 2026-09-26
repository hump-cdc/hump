import { type ComponentProps, type FC, type Ref, useMemo } from 'react'

import { Flow, useFlowSelection, useSelectionHandlers } from '@/shared/ui/flow'

import { nodeTypes } from './flow-summary.consts'
import { partitionChildren, useSummaryEdgeAnimation } from './flow-summary.helpers'
import type {
    ClearSelectionHandler,
    EdgeSelectHandler,
    FlowSummaryRef,
    SinkNodeSelectHandler,
    TableNodeSelectHandler,
} from './flow-summary.types'
import type { SinkNodeData } from './node.sink'

export { TableNode as FlowSummaryTable, type TableNodeData } from '@/shared/ui/flow'

export type {
    ClearSelectionHandler,
    EdgeSelectHandler,
    FlowSummaryRef,
    SinkNodeSelectHandler,
    TableNodeSelectHandler,
} from './flow-summary.types'
export { SinkNode as FlowSummarySink, type SinkNodeData } from './node.sink'

type FlowSummaryProps = Omit<ComponentProps<typeof Flow>, 'nodes' | 'selected' | 'ref'> & {
    /** Id of the currently selected node or edge. */
    selected?: string
    onTableNodeSelect?: TableNodeSelectHandler
    onSinkNodeSelect?: SinkNodeSelectHandler
    onEdgeSelect?: EdgeSelectHandler
    onClearSelection?: ClearSelectionHandler
    ref?: Ref<FlowSummaryRef>
}

/**
 * Summary topology: source tables (left, with insert/update/delete/truncate counts)
 * wired to one aggregate node per sink (right). Edge weight comes from each edge's
 * `style` (`opacity`/`strokeWidth`); `ref.animate` flashes a route.
 */
export const FlowSummary: FC<FlowSummaryProps> = ({
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

    const animatedEdges = useSummaryEdgeAnimation(edges, ref)

    const { selectedNodes, selectedEdges } = useFlowSelection(nodes, edges, selected, animatedEdges)

    const { handleNodeClick, handleEdgeClick, handlePaneClick } = useSelectionHandlers<SinkNodeData>({
        selected,
        onTableNodeSelect,
        onSinkNodeSelect,
        onEdgeSelect,
        onClearSelection,
    })

    return (
        <Flow
            nodes={selectedNodes}
            edges={selectedEdges}
            nodeTypes={nodeTypes}
            onNodeClick={handleNodeClick}
            onEdgeClick={handleEdgeClick}
            onPaneClick={handlePaneClick}
            revealId={selected}
            {...props}
        >
            {rest}
        </Flow>
    )
}
