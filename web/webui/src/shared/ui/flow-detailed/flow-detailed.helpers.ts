import {
    Children,
    isValidElement,
    type ReactNode,
    type Ref,
    useCallback,
    useEffect,
    useImperativeHandle,
    useMemo,
    useRef,
    useState,
} from 'react'

import type { Edge, EdgeMouseHandler, Node, NodeMouseHandler } from '@xyflow/react'

import {
    DEFAULT_ANIMATE_DURATION,
    EDGE_Z_INDEX,
    GHOST_GAP,
    GHOST_HEIGHT,
    LABEL_OFFSET_Y,
    SINK_CHROME_HEIGHT,
    SINK_COLUMN_X,
    SINK_GAP,
    TABLE_COLUMN_X,
    TABLE_GAP,
    TABLE_HEIGHT,
    TARGET_GAP,
    TARGET_ROW_HEIGHT,
} from './flow-detailed.consts'
import { sinkTargetKey } from './flow-detailed.context'
import type {
    ClearSelectionHandler,
    EdgeSelectHandler,
    FlowDetailedRef,
    SinkNodeSelectHandler,
    SinkTargetSelectHandler,
    TableNodeSelectHandler,
} from './flow-detailed.types'
import type { FlowNodeBase } from './node'
import type { SinkNodeData, SinkTarget } from './node.sink'
import type { TableNodeData } from './node.table'

type NodeSpec = { id: string; type: string; data: Record<string, unknown> }

/** A node spec with its estimated rendered height and the gap to leave before it. */
type StackItem = NodeSpec & { height: number; gap: number }

/**
 * Stack `items` top-to-bottom as a column at `x`, each preceded by its `gap` (ignored for the
 * first), centering the whole column vertically around y=0.
 */
const stackColumn = (items: StackItem[], x: number): Node[] => {
    const total = items.reduce((sum, item, i) => sum + item.height + (i > 0 ? item.gap : 0), 0)
    let y = -total / 2

    return items.map(({ id, type, data, height, gap }, i) => {
        if (i > 0) {
            y += gap
        }
        const node: Node = { id, type, position: { x, y }, data }
        y += height
        return node
    })
}

/** Estimated sink height: fixed chrome plus one row per target. */
const sinkHeight = (data: Record<string, unknown>): number => {
    const count = (data as SinkNodeData).targets.length
    return SINK_CHROME_HEIGHT + count * TARGET_ROW_HEIGHT + Math.max(0, count - 1) * TARGET_GAP
}

/** Non-interactive caption node sitting `LABEL_OFFSET_Y` above the first node of `column`. */
const columnLabel = (id: string, text: string, column: Node[], x: number): Node[] =>
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
 * Turn declarative node-config children (`<FlowDetailedTable>` / `<FlowDetailedSink>` /
 * `<FlowDetailedGhost>`, tagged with `flowNodeType`) into React Flow nodes: tables form a
 * left column, sinks then ghosts stack in a right column, each in declaration order —
 * callers don't supply coordinates — and each column gets a caption. Other children are
 * returned as-is so they pass through to `<ReactFlow>` as overlays.
 */
export const partitionChildren = (children: ReactNode): { nodes: Node[]; rest: ReactNode[] } => {
    const tables: StackItem[] = []
    const sinks: StackItem[] = []
    const ghosts: StackItem[] = []
    const rest: ReactNode[] = []

    for (const child of Children.toArray(children)) {
        if (!isValidElement(child)) {
            rest.push(child)
            continue
        }

        const flowNodeType = (child.type as { flowNodeType?: string }).flowNodeType
        const { id, ...data } = child.props as FlowNodeBase & Record<string, unknown>

        if (flowNodeType === 'table') {
            tables.push({ id, type: flowNodeType, data, height: TABLE_HEIGHT, gap: TABLE_GAP })
        } else if (flowNodeType === 'sink') {
            sinks.push({ id, type: flowNodeType, data, height: sinkHeight(data), gap: SINK_GAP })
        } else if (flowNodeType === 'ghost') {
            ghosts.push({ id, type: flowNodeType, data, height: GHOST_HEIGHT, gap: GHOST_GAP })
        } else {
            rest.push(child)
        }
    }

    // The first ghost after the sinks keeps the tight sink spacing; later ghosts use their own gap.
    if (sinks.length > 0 && ghosts.length > 0) {
        ghosts[0] = { ...ghosts[0], gap: SINK_GAP }
    }

    const tableNodes = stackColumn(tables, TABLE_COLUMN_X)
    const sinkNodes = stackColumn([...sinks, ...ghosts], SINK_COLUMN_X).map((node) =>
        node.type === 'ghost' ? { ...node, selectable: false } : node,
    )

    const nodes = [
        ...tableNodes,
        ...sinkNodes,
        ...columnLabel('label-tables', 'TABLES', tableNodes, TABLE_COLUMN_X),
        ...columnLabel('label-sinks', 'SINKS', sinkNodes, SINK_COLUMN_X),
    ]

    return { nodes, rest }
}

/**
 * Drive marching-ants edge animation: returns the set of currently-animated edge ids
 * and wires the imperative `animate(tableId, sinkId, targetId?, durationMs?)` handle onto `ref`.
 * Each `animate` call flags the matching edge, then clears it after `durationMs`; a
 * repeat call resets the timer. All pending timers are cleared on unmount.
 */
export const useEdgeAnimation = (
    edges: Edge[] | undefined,
    ref: Ref<FlowDetailedRef> | undefined,
): ReadonlySet<string> => {
    const [animatedIds, setAnimatedIds] = useState<ReadonlySet<string>>(() => new Set())

    const edgesRef = useRef(edges)
    edgesRef.current = edges

    const timersRef = useRef(new Map<string, ReturnType<typeof setTimeout>>())

    const animate = useCallback<FlowDetailedRef['animate']>(
        (tableId, sinkId, targetId, durationMs = DEFAULT_ANIMATE_DURATION) => {
            const edge = edgesRef.current?.find(
                (e) => e.source === tableId && e.target === sinkId && (e.targetHandle ?? undefined) === targetId,
            )
            if (!edge) {
                return
            }

            const { id } = edge
            setAnimatedIds((prev) => new Set(prev).add(id))

            const timers = timersRef.current
            const existing = timers.get(id)
            if (existing) {
                clearTimeout(existing)
            }

            timers.set(
                id,
                setTimeout(() => {
                    timers.delete(id)
                    setAnimatedIds((prev) => {
                        const next = new Set(prev)
                        next.delete(id)
                        return next
                    })
                }, durationMs),
            )
        },
        [],
    )

    useImperativeHandle(ref, () => ({ animate }), [animate])

    useEffect(() => {
        const timers = timersRef.current
        return () => {
            for (const timer of timers.values()) {
                clearTimeout(timer)
            }
            timers.clear()
        }
    }, [])

    return animatedIds
}

/**
 * Decorate `nodes`/`edges` for ReactFlow: flag the `selected` item, lift edges above nodes
 * and mark edges in `animatedIds` as `animated`. Memoized so identity is stable across renders.
 */
export const useFlowSelection = (
    nodes: Node[],
    edges: Edge[] | undefined,
    selected: string | undefined,
    animatedIds: ReadonlySet<string>,
): { selectedNodes: Node[]; selectedEdges: Edge[] | undefined } => {
    const selectedNodes = useMemo(
        () => nodes.map((node) => ({ ...node, selected: node.id === selected })),
        [nodes, selected],
    )

    const selectedEdges = useMemo(
        () =>
            edges?.map((edge) => ({
                ...edge,
                zIndex: EDGE_Z_INDEX,
                selected: edge.id === selected,
                animated: animatedIds.has(edge.id),
            })),
        [edges, selected, animatedIds],
    )

    return { selectedNodes, selectedEdges }
}

/** Selection callbacks consumed by {@link useSelectionHandlers}. */
type SelectionHandlerOptions = {
    selected: string | undefined
    onTableNodeSelect?: TableNodeSelectHandler
    onSinkNodeSelect?: SinkNodeSelectHandler
    onSinkTargetSelect?: SinkTargetSelectHandler
    onEdgeSelect?: EdgeSelectHandler
    onClearSelection?: ClearSelectionHandler
}

/**
 * Build the node/target/edge/pane click handlers that translate clicks into selection
 * callbacks. Clicking the already-`selected` item (or the empty pane) clears the selection.
 */
export const useSelectionHandlers = ({
    selected,
    onTableNodeSelect,
    onSinkNodeSelect,
    onSinkTargetSelect,
    onEdgeSelect,
    onClearSelection,
}: SelectionHandlerOptions): {
    handleNodeClick: NodeMouseHandler
    handleTargetClick: (sinkId: string, target: SinkTarget) => void
    handleEdgeClick: EdgeMouseHandler
    handlePaneClick: () => void
} => {
    const handleNodeClick = useCallback<NodeMouseHandler>(
        (_event, node) => {
            if (node.type === 'label' || node.type === 'ghost') {
                return
            }
            if (node.id === selected) {
                onClearSelection?.()
                return
            }
            if (node.type === 'table') {
                onTableNodeSelect?.(node.id, node.data as TableNodeData)
            } else if (node.type === 'sink') {
                onSinkNodeSelect?.(node.id, node.data as SinkNodeData)
            }
        },
        [selected, onTableNodeSelect, onSinkNodeSelect, onClearSelection],
    )

    const handleTargetClick = useCallback(
        (sinkId: string, target: SinkTarget) => {
            if (sinkTargetKey(sinkId, target.id) === selected) {
                onClearSelection?.()
                return
            }
            onSinkTargetSelect?.(sinkId, target)
        },
        [selected, onSinkTargetSelect, onClearSelection],
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

    return { handleNodeClick, handleTargetClick, handleEdgeClick, handlePaneClick }
}
