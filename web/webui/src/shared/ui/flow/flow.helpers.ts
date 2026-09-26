import { Children, isValidElement, type ReactNode, useCallback, useEffect, useMemo, useRef, useState } from 'react'

import type { Dimensions, Edge, EdgeMouseHandler, Node, NodeMouseHandler, OnNodesChange } from '@xyflow/react'

import { DEFAULT_ANIMATE_DURATION, EDGE_DASH_PERIOD, LABEL_OFFSET_Y } from './flow.consts'
import type {
    ClearSelectionHandler,
    EdgeSelectHandler,
    FlowNodeSpec,
    TableNodeSelectHandler,
    ViewportInsets,
} from './flow.types'
import type { FlowNodeBase } from './node'
import type { TableNodeData } from './node.table'

/** Non-interactive caption node sitting `LABEL_OFFSET_Y` above the first node of `column`. */
export const columnLabel = (id: string, text: string, column: Node[], x: number): Node[] =>
    column.length === 0
        ? []
        : [
              {
                  id,
                  type: 'label',
                  position: { x, y: Math.min(...column.map((node) => node.position.y)) - LABEL_OFFSET_Y },
                  data: { text },
                  selectable: false,
                  focusable: false,
              },
          ]

/**
 * Split children into declarative node-config elements (tagged with `flowNodeType`), resolved
 * to node specs in declaration order, and everything else, returned as-is so it passes through
 * to `<ReactFlow>` as overlays. Callers lay the specs out into columns.
 */
export const collectNodeSpecs = (children: ReactNode): { specs: FlowNodeSpec[]; rest: ReactNode[] } => {
    const specs: FlowNodeSpec[] = []
    const rest: ReactNode[] = []

    for (const child of Children.toArray(children)) {
        const flowNodeType = isValidElement(child) ? (child.type as { flowNodeType?: string }).flowNodeType : undefined

        if (!isValidElement(child) || !flowNodeType) {
            rest.push(child)
            continue
        }

        const { id, ...data } = child.props as FlowNodeBase & Record<string, unknown>
        specs.push({ id, type: flowNodeType, data })
    }

    return { specs, rest }
}

/**
 * Drive marching-ants edge animation: returns the currently-animated edge ids, each mapped to its
 * `animation-delay` (ms), and `animateEdge(id, durationMs?)`, which flags the edge, then clears it
 * after `durationMs`; a repeat call resets the timer and keeps the running animation. The delay
 * phase-locks every start to a shared clock, so a restarted edge picks up where its dashes would
 * be had they never stopped. All pending timers are cleared on unmount.
 */
export const useEdgeAnimation = (): {
    animatedEdges: ReadonlyMap<string, number>
    animateEdge: (id: string, durationMs?: number) => void
} => {
    const [animatedEdges, setAnimatedEdges] = useState<ReadonlyMap<string, number>>(() => new Map())

    const timersRef = useRef(new Map<string, ReturnType<typeof setTimeout>>())

    const animateEdge = useCallback((id: string, durationMs = DEFAULT_ANIMATE_DURATION) => {
        const delay = -(performance.now() % EDGE_DASH_PERIOD)
        setAnimatedEdges((prev) => (prev.has(id) ? prev : new Map(prev).set(id, delay)))

        const timers = timersRef.current
        const existing = timers.get(id)
        if (existing) {
            clearTimeout(existing)
        }

        timers.set(
            id,
            setTimeout(() => {
                timers.delete(id)
                setAnimatedEdges((prev) => {
                    const next = new Map(prev)
                    next.delete(id)
                    return next
                })
            }, durationMs),
        )
    }, [])

    useEffect(() => {
        const timers = timersRef.current
        return () => {
            for (const timer of timers.values()) {
                clearTimeout(timer)
            }
            timers.clear()
        }
    }, [])

    return { animatedEdges, animateEdge }
}

/**
 * Decorate `nodes`/`edges` for ReactFlow: flag the `selected` item, mark edges in `animatedEdges`
 * as `animated` with their phase-locking `animationDelay` and, when given, apply `edgeZIndex`. Memoized so identity is stable across renders.
 */
export const useFlowSelection = (
    nodes: Node[],
    edges: Edge[] | undefined,
    selected: string | undefined,
    animatedEdges: ReadonlyMap<string, number>,
    edgeZIndex?: number,
): { selectedNodes: Node[]; selectedEdges: Edge[] | undefined } => {
    const selectedNodes = useMemo(
        () => nodes.map((node) => ({ ...node, selected: node.id === selected })),
        [nodes, selected],
    )

    const selectedEdges = useMemo(
        () =>
            edges?.map((edge) => {
                const delay = animatedEdges.get(edge.id)
                return {
                    ...edge,
                    ...(edgeZIndex === undefined ? {} : { zIndex: edgeZIndex }),
                    ...(delay === undefined ? {} : { style: { ...edge.style, animationDelay: `${delay}ms` } }),
                    selected: edge.id === selected,
                    animated: delay !== undefined,
                }
            }),
        [edges, selected, animatedEdges, edgeZIndex],
    )

    return { selectedNodes, selectedEdges }
}

/** Selection callbacks consumed by {@link useSelectionHandlers}; `S` is the canvas's sink node data. */
type SelectionHandlerOptions<S> = {
    selected: string | undefined
    onTableNodeSelect?: TableNodeSelectHandler
    onSinkNodeSelect?: (id: string, data: S) => void
    onEdgeSelect?: EdgeSelectHandler
    onClearSelection?: ClearSelectionHandler
}

/**
 * Build the node/edge/pane click handlers that translate ReactFlow clicks into selection
 * callbacks. Only table and sink nodes are selectable. Clicking the already-`selected` item
 * (or the empty pane) clears the selection.
 */
export const useSelectionHandlers = <S>({
    selected,
    onTableNodeSelect,
    onSinkNodeSelect,
    onEdgeSelect,
    onClearSelection,
}: SelectionHandlerOptions<S>): {
    handleNodeClick: NodeMouseHandler
    handleEdgeClick: EdgeMouseHandler
    handlePaneClick: () => void
} => {
    const handleNodeClick = useCallback<NodeMouseHandler>(
        (_event, node) => {
            if (node.type !== 'table' && node.type !== 'sink') {
                return
            }
            if (node.id === selected) {
                onClearSelection?.()
                return
            }
            if (node.type === 'table') {
                onTableNodeSelect?.(node.id, node.data as TableNodeData)
            } else {
                onSinkNodeSelect?.(node.id, node.data as S)
            }
        },
        [selected, onTableNodeSelect, onSinkNodeSelect, onClearSelection],
    )

    const handleEdgeClick = useCallback<EdgeMouseHandler>(
        (_event, edge) => {
            if (edge.id === selected) {
                onClearSelection?.()
                return
            }
            onEdgeSelect?.(edge.id, { source: edge.source, target: edge.target, targetHandle: edge.targetHandle })
        },
        [selected, onEdgeSelect, onClearSelection],
    )

    const handlePaneClick = useCallback(() => {
        onClearSelection?.()
    }, [onClearSelection])

    return { handleNodeClick, handleEdgeClick, handlePaneClick }
}

/**
 * Carry each node's measured size onto `nodes`. React Flow drops a node's handle bounds when a
 * new node object arrives without `measured`, which unmounts its edges for a frame and restarts
 * their animation; rebuilt nodes (e.g. on every metrics refresh) would otherwise flicker edges.
 * Returns the patched nodes and an `onNodesChange` that records sizes, then calls `onNodesChange`.
 */
export const useMeasuredNodes = (
    nodes: Node[] | undefined,
    onNodesChange: OnNodesChange | undefined,
): { measuredNodes: Node[] | undefined; handleNodesChange: OnNodesChange } => {
    const [sizes, setSizes] = useState<ReadonlyMap<string, Dimensions>>(() => new Map())

    const handleNodesChange = useCallback<OnNodesChange>(
        (changes) => {
            const measured = changes.flatMap((change) =>
                change.type === 'dimensions' && change.dimensions ? [[change.id, change.dimensions] as const] : [],
            )
            if (measured.length > 0) {
                setSizes((prev) => new Map([...prev, ...measured]))
            }
            onNodesChange?.(changes)
        },
        [onNodesChange],
    )

    const measuredNodes = useMemo(
        () =>
            nodes?.map((node) => {
                const size = sizes.get(node.id)
                return size && !node.measured ? { ...node, measured: size } : node
            }),
        [nodes, sizes],
    )

    return { measuredNodes, handleNodesChange }
}

/** Axis-aligned rectangle in flow coordinates. */
export type FlowRect = { x: number; y: number; width: number; height: number }

/**
 * Viewport translation that brings `rect` fully into the unobstructed canvas area (the canvas
 * `size` minus `insets`, shrunk by `margin`) at the current zoom, moving the least on each axis.
 * A rect larger than the area is aligned to its left/top edge. Returns `null` when no move is needed.
 */
export const getRevealViewport = (
    rect: FlowRect,
    viewport: { x: number; y: number; zoom: number },
    size: { width: number; height: number },
    insets: ViewportInsets,
    margin: number,
): { x: number; y: number } | null => {
    const shift = (start: number, length: number, min: number, max: number): number => {
        if (start < min || length > max - min) {
            return min - start
        }
        return start + length > max ? max - (start + length) : 0
    }

    const dx = shift(
        rect.x * viewport.zoom + viewport.x,
        rect.width * viewport.zoom,
        insets.left + margin,
        size.width - insets.right - margin,
    )
    const dy = shift(
        rect.y * viewport.zoom + viewport.y,
        rect.height * viewport.zoom,
        insets.top + margin,
        size.height - insets.bottom - margin,
    )

    return dx === 0 && dy === 0 ? null : { x: viewport.x + dx, y: viewport.y + dy }
}
