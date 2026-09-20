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
    LABEL_OFFSET_Y,
    SINK_COLUMN_X,
    SINK_ROW_STEP,
    TABLE_COLUMN_X,
    TABLE_ROW_STEP,
} from './flow-summary.consts'
import type {
    ClearSelectionHandler,
    EdgeSelectHandler,
    FlowSummaryRef,
    SinkNodeSelectHandler,
    TableNodeSelectHandler,
} from './flow-summary.types'
import type { FlowNodeBase } from './node'
import type { SinkNodeData } from './node.sink'
import type { TableNodeData } from './node.table'

/** Place `ids` as a column at `x`, stepping by `rowStep` and centered vertically around y=0 */
const layoutColumn = (
    ids: { id: string; type: string; data: Record<string, unknown> }[],
    x: number,
    rowStep: number,
): Node[] => {
    const span = (ids.length - 1) * rowStep
    return ids.map(({ id, type, data }, i) => ({
        id,
        type,
        position: { x, y: i * rowStep - span / 2 },
        data,
    }))
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
 * Turn declarative node-config children (`<FlowSummaryTable>` / `<FlowSummarySink>`, tagged with
 * `flowNodeType`) into React Flow nodes, arranging tables into a left column and sinks
 * into a right column in declaration order — callers don't supply coordinates — and caption
 * each column. Other
 * children are returned as-is so they pass through to `<ReactFlow>` as overlays.
 */
export const partitionChildren = (children: ReactNode): { nodes: Node[]; rest: ReactNode[] } => {
    const tables: { id: string; type: string; data: Record<string, unknown> }[] = []
    const sinks: { id: string; type: string; data: Record<string, unknown> }[] = []
    const rest: ReactNode[] = []

    for (const child of Children.toArray(children)) {
        if (!isValidElement(child)) {
            rest.push(child)
            continue
        }

        const flowNodeType = (child.type as { flowNodeType?: string }).flowNodeType
        const { id, ...data } = child.props as FlowNodeBase & Record<string, unknown>

        if (flowNodeType === 'table') {
            tables.push({ id, type: flowNodeType, data })
        } else if (flowNodeType === 'sink') {
            sinks.push({ id, type: flowNodeType, data })
        } else {
            rest.push(child)
        }
    }

    const tableNodes = layoutColumn(tables, TABLE_COLUMN_X, TABLE_ROW_STEP)
    const sinkNodes = layoutColumn(sinks, SINK_COLUMN_X, SINK_ROW_STEP)

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
 * and wires the imperative `animate(tableId, sinkId, durationMs?)` handle onto `ref`.
 * Each `animate` call flags the matching edge, then clears it after `durationMs`; a
 * repeat call resets the timer. All pending timers are cleared on unmount.
 */
export const useEdgeAnimation = (
    edges: Edge[] | undefined,
    ref: Ref<FlowSummaryRef> | undefined,
): ReadonlySet<string> => {
    const [animatedIds, setAnimatedIds] = useState<ReadonlySet<string>>(() => new Set())

    const edgesRef = useRef(edges)
    edgesRef.current = edges

    const timersRef = useRef(new Map<string, ReturnType<typeof setTimeout>>())

    const animate = useCallback<FlowSummaryRef['animate']>((tableId, sinkId, durationMs = DEFAULT_ANIMATE_DURATION) => {
        const edge = edgesRef.current?.find((e) => e.source === tableId && e.target === sinkId)
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
    }, [])

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
 * Decorate `nodes`/`edges` for ReactFlow: flag the `selected` item and mark edges in
 * `animatedIds` as `animated`. Memoized so identity is stable across renders.
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
        () => edges?.map((edge) => ({ ...edge, selected: edge.id === selected, animated: animatedIds.has(edge.id) })),
        [edges, selected, animatedIds],
    )

    return { selectedNodes, selectedEdges }
}

/** Selection callbacks consumed by {@link useSelectionHandlers}. */
type SelectionHandlerOptions = {
    selected: string | undefined
    onTableNodeSelect?: TableNodeSelectHandler
    onSinkNodeSelect?: SinkNodeSelectHandler
    onEdgeSelect?: EdgeSelectHandler
    onClearSelection?: ClearSelectionHandler
}

/**
 * Build the node/edge/pane click handlers that translate ReactFlow clicks into selection
 * callbacks. Clicking the already-`selected` item (or the empty pane) clears the selection.
 */
export const useSelectionHandlers = ({
    selected,
    onTableNodeSelect,
    onSinkNodeSelect,
    onEdgeSelect,
    onClearSelection,
}: SelectionHandlerOptions): {
    handleNodeClick: NodeMouseHandler
    handleEdgeClick: EdgeMouseHandler
    handlePaneClick: () => void
} => {
    const handleNodeClick = useCallback<NodeMouseHandler>(
        (_event, node) => {
            if (node.type === 'label') {
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

    const handleEdgeClick = useCallback<EdgeMouseHandler>(
        (_event, edge) => {
            if (edge.id === selected) {
                onClearSelection?.()
                return
            }
            onEdgeSelect?.(edge.id, { source: edge.source, target: edge.target })
        },
        [selected, onEdgeSelect, onClearSelection],
    )

    const handlePaneClick = useCallback(() => {
        onClearSelection?.()
    }, [onClearSelection])

    return { handleNodeClick, handleEdgeClick, handlePaneClick }
}
