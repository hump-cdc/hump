import { type FC, type RefObject, useEffect, useRef } from 'react'

import { type Handle, type InternalNode, type ReactFlowState, useReactFlow, useStoreApi } from '@xyflow/react'

import { REVEAL_DURATION, REVEAL_MARGIN } from './flow.consts'
import { type FlowRect, getRevealViewport } from './flow.helpers'
import type { ViewportInsets } from './flow.types'

type FlowRevealProps = {
    /** Id of the node or edge to bring into view. */
    revealId: string | undefined
    /** Selection the reveal was requested for; a new key re-reveals even when it resolves to the same `revealId`. */
    revealKey: string | undefined
    insets: ViewportInsets
    /** Set by the canvas when the user pans or zooms; stops re-revealing the current id on inset changes. */
    userMovedRef: RefObject<boolean>
}

/**
 * Pan (at the current zoom, animated) so the `revealId` node or edge sits fully inside the canvas
 * area left uncovered by `insets` and the zoom controls on the left. Re-runs when the insets change (e.g. the inspector opens) until
 * the user moves the viewport. Deferred a frame so insets settling right after a selection yield a
 * single pan. Renders nothing; must sit inside `<ReactFlow>`.
 */
export const FlowReveal: FC<FlowRevealProps> = ({
    revealId,
    revealKey,
    insets: { top, right, bottom, left },
    userMovedRef,
}) => {
    const store = useStoreApi()
    const { setViewport } = useReactFlow()
    const revealedKeyRef = useRef<string | undefined>(undefined)

    useEffect(() => {
        if (revealKey !== revealedKeyRef.current) {
            revealedKeyRef.current = revealKey
            userMovedRef.current = false
        }
        if (revealId === undefined || userMovedRef.current) {
            return
        }

        const frame = requestAnimationFrame(() => {
            const state = store.getState()
            const rect = getElementRect(state, revealId)
            if (!rect) {
                return
            }

            const [x, y, zoom] = state.transform
            const next = getRevealViewport(
                rect,
                { x, y, zoom },
                { width: state.width, height: state.height },
                { top, right, bottom, left: Math.max(left, getControlsRight(state.domNode)) },
                REVEAL_MARGIN,
            )
            if (next) {
                void setViewport({ ...next, zoom }, { duration: REVEAL_DURATION })
            }
        })

        return () => cancelAnimationFrame(frame)
    }, [revealId, revealKey, top, right, bottom, left, store, setViewport, userMovedRef])

    return null
}

/** Right edge (px, canvas-relative) of the zoom controls panel, or 0 when it isn't rendered. */
const getControlsRight = (domNode: ReactFlowState['domNode']): number => {
    const controls = domNode?.querySelector('.react-flow__controls')
    return domNode && controls ? controls.getBoundingClientRect().right - domNode.getBoundingClientRect().left : 0
}

/** Bounds of a measured node, or of the straight span between an edge's two handles; `null` if not measured yet. */
const getElementRect = (state: ReactFlowState, id: string): FlowRect | null => {
    const node = state.nodeLookup.get(id)
    if (node) {
        const { width, height } = node.measured
        return width === undefined || height === undefined
            ? null
            : { ...node.internals.positionAbsolute, width, height }
    }

    const edge = state.edgeLookup.get(id)
    if (!edge) {
        return null
    }

    const source = getHandleCenter(state.nodeLookup.get(edge.source), 'source', edge.sourceHandle)
    const target = getHandleCenter(state.nodeLookup.get(edge.target), 'target', edge.targetHandle)
    if (!source || !target) {
        return null
    }

    // A left-to-right bezier's control points share its endpoints' y, so the curve stays within this box.
    return {
        x: Math.min(source.x, target.x),
        y: Math.min(source.y, target.y),
        width: Math.abs(target.x - source.x),
        height: Math.abs(target.y - source.y),
    }
}

/** Absolute center of a node's handle (by id, else its first one of `type`); `null` if not measured yet. */
const getHandleCenter = (
    node: InternalNode | undefined,
    type: 'source' | 'target',
    handleId: string | null | undefined,
): { x: number; y: number } | null => {
    const handles: Handle[] | null | undefined = node?.internals.handleBounds?.[type]
    const handle = handles?.find((candidate) => !handleId || candidate.id === handleId)
    if (!node || !handle) {
        return null
    }

    return {
        x: node.internals.positionAbsolute.x + handle.x + handle.width / 2,
        y: node.internals.positionAbsolute.y + handle.y + handle.height / 2,
    }
}
