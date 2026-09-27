import { type ComponentProps, type FC, type ReactNode, useCallback, useRef } from 'react'

import { Background, BackgroundVariant, type OnMoveStart, ReactFlow, ReactFlowProvider } from '@xyflow/react'

import { cn } from '@/shared/utils'

import { FlowControls } from './flow.controls'
import { useMeasuredNodes } from './flow.helpers'
import { FlowReveal } from './flow.reveal'
import { useFlowTransition } from './flow.transition'
import type { ViewportInsets } from './flow.types'

export { ControlButton as FlowControlButton } from '@xyflow/react'

export { DEFAULT_ANIMATE_DURATION, LABEL_OFFSET_Y, SINK_COLUMN_X, TABLE_COLUMN_X } from './flow.consts'
export {
    collectNodeSpecs,
    columnLabel,
    useEdgeAnimation,
    useFlowSelection,
    useMeasuredNodes,
    useSelectionHandlers,
} from './flow.helpers'
export type {
    ClearSelectionHandler,
    EdgeSelectHandler,
    FlowNodeSpec,
    TableNodeSelectHandler,
    ViewportInsets,
} from './flow.types'
export type { FlowNodeBase, FlowNodeConfig } from './node'
export { LabelNodeView } from './node.label'
export { TableNode, type TableNodeData, TableNodeView } from './node.table'
export { SinkIcon, SinkIconTile } from './sink-icon'

import '@xyflow/react/dist/style.css'
import './flow.css'

type FlowProps = ComponentProps<typeof ReactFlow> & {
    /** Id of a node or edge to bring into view (animated pan) whenever it changes. */
    revealId?: string
    /** Selection behind `revealId` (defaults to it); a change re-reveals even if `revealId` stays the same. */
    revealKey?: string
    /** Canvas margins covered by floating overlays, kept clear when revealing `revealId`. */
    viewportInsets?: ViewportInsets
    /** Extra `FlowControlButton`s appended to the zoom controls card. */
    controls?: ReactNode
}

const NO_INSETS: ViewportInsets = { top: 0, right: 0, bottom: 0, left: 0 }

/**
 * Read-only React Flow canvas shared by the topology views: fixed (non-draggable,
 * non-connectable) nodes fitted into view (zoom capped at 100%) over a dotted background with
 * animated zoom controls (plus any `controls`); layout changes animate (nodes glide, added and removed nodes and edges fade); `revealId` pans to keep the selection clear of overlays. Children render as overlays inside the canvas. Measured node sizes survive rebuilt `nodes`,
 * so edges stay mounted (and keep animating) across data refreshes.
 */
export const Flow: FC<FlowProps> = ({
    className,
    children,
    nodes,
    edges,
    onNodesChange,
    onMoveStart,
    revealId,
    revealKey = revealId,
    viewportInsets = NO_INSETS,
    controls,
    ...props
}) => {
    const transition = useFlowTransition(nodes, edges)
    const { measuredNodes, handleNodesChange } = useMeasuredNodes(transition.nodes, onNodesChange)

    // Programmatic moves (reveal, zoom controls) carry no source event; only user gestures do.
    const userMovedRef = useRef(false)
    const handleMoveStart = useCallback<OnMoveStart>(
        (event, viewport) => {
            if (event) {
                userMovedRef.current = true
            }
            onMoveStart?.(event, viewport)
        },
        [onMoveStart],
    )

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
                    maxZoom={1}
                    proOptions={{ hideAttribution: true }}
                    {...props}
                    nodes={measuredNodes}
                    edges={transition.edges}
                    onNodesChange={handleNodesChange}
                    onMoveStart={handleMoveStart}
                    colorMode="light"
                >
                    <Background variant={BackgroundVariant.Dots} gap={20} size={1} />
                    <FlowControls insets={viewportInsets}>{controls}</FlowControls>
                    <FlowReveal
                        revealId={revealId}
                        revealKey={revealKey}
                        insets={viewportInsets}
                        userMovedRef={userMovedRef}
                    />
                    {children}
                </ReactFlow>
            </div>
        </ReactFlowProvider>
    )
}
