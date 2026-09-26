import { type ReactNode, type Ref, useCallback, useImperativeHandle, useRef } from 'react'

import type { Edge, Node } from '@xyflow/react'

import {
    type ClearSelectionHandler,
    collectNodeSpecs,
    columnLabel,
    type FlowNodeSpec,
    SINK_COLUMN_X,
    TABLE_COLUMN_X,
    useEdgeAnimation,
} from '@/shared/ui/flow'

import {
    GHOST_GAP,
    GHOST_HEIGHT,
    SINK_CHROME_HEIGHT,
    SINK_GAP,
    TABLE_GAP,
    TABLE_HEIGHT,
    TARGET_GAP,
    TARGET_ROW_HEIGHT,
} from './flow-detailed.consts'
import { sinkTargetKey } from './flow-detailed.context'
import type { FlowDetailedRef, SinkTargetSelectHandler } from './flow-detailed.types'
import type { SinkNodeData, SinkTarget } from './node.sink'

/** A node spec with its estimated rendered height and the gap to leave before it. */
type StackItem = FlowNodeSpec & { height: number; gap: number }

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

/**
 * Turn declarative node-config children (`<FlowDetailedTable>` / `<FlowDetailedSink>` /
 * `<FlowDetailedGhost>`) into React Flow nodes: tables form a left column, sinks then ghosts
 * stack in a right column, each in declaration order — callers don't supply coordinates — and
 * each column gets a caption. Other children are returned as-is so they pass through to
 * `<ReactFlow>` as overlays.
 */
export const partitionChildren = (children: ReactNode): { nodes: Node[]; rest: ReactNode[] } => {
    const { specs, rest } = collectNodeSpecs(children)

    const tables: StackItem[] = []
    const sinks: StackItem[] = []
    const ghosts: StackItem[] = []

    for (const spec of specs) {
        if (spec.type === 'table') {
            tables.push({ ...spec, height: TABLE_HEIGHT, gap: TABLE_GAP })
        } else if (spec.type === 'sink') {
            sinks.push({ ...spec, height: sinkHeight(spec.data), gap: SINK_GAP })
        } else if (spec.type === 'ghost') {
            ghosts.push({ ...spec, height: GHOST_HEIGHT, gap: GHOST_GAP })
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
 * Drive marching-ants edge animation: returns the currently-animated edge ids, mapped to their
 * `animation-delay`, and wires the imperative `animate(tableId, sinkId, targetId?, durationMs?)`
 * handle onto `ref`.
 */
export const useDetailedEdgeAnimation = (
    edges: Edge[] | undefined,
    ref: Ref<FlowDetailedRef> | undefined,
): ReadonlyMap<string, number> => {
    const { animatedEdges, animateEdge } = useEdgeAnimation()

    const edgesRef = useRef(edges)
    edgesRef.current = edges

    const animate = useCallback<FlowDetailedRef['animate']>(
        (tableId, sinkId, targetId, durationMs) => {
            const edge = edgesRef.current?.find(
                (e) => e.source === tableId && e.target === sinkId && (e.targetHandle ?? undefined) === targetId,
            )
            if (edge) {
                animateEdge(edge.id, durationMs)
            }
        },
        [animateEdge],
    )

    useImperativeHandle(ref, () => ({ animate }), [animate])

    return animatedEdges
}

/**
 * Build the target-row click handler: selects the target, or clears the selection when the
 * target is already `selected`.
 */
export const useTargetClickHandler = (
    selected: string | undefined,
    onSinkTargetSelect: SinkTargetSelectHandler | undefined,
    onClearSelection: ClearSelectionHandler | undefined,
): ((sinkId: string, target: SinkTarget) => void) =>
    useCallback(
        (sinkId: string, target: SinkTarget) => {
            if (sinkTargetKey(sinkId, target.id) === selected) {
                onClearSelection?.()
                return
            }
            onSinkTargetSelect?.(sinkId, target)
        },
        [selected, onSinkTargetSelect, onClearSelection],
    )

/**
 * Id of the canvas element to reveal for `selected`: a selected target row resolves to its sink
 * node (the whole card is revealed); a node or edge id passes through.
 */
export const getRevealId = (nodes: Node[], selected: string | undefined): string | undefined =>
    nodes.find(
        (node) =>
            node.type === 'sink' &&
            (node.data as SinkNodeData).targets.some((target) => sinkTargetKey(node.id, target.id) === selected),
    )?.id ?? selected
