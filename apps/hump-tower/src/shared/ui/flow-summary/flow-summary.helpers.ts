import { type ReactNode, type Ref, useCallback, useImperativeHandle, useRef } from 'react'

import type { Edge, Node } from '@xyflow/react'

import {
    collectNodeSpecs,
    columnLabel,
    type FlowNodeSpec,
    SINK_COLUMN_X,
    TABLE_COLUMN_X,
    useEdgeAnimation,
} from '@/shared/ui/flow'

import { SINK_ROW_STEP, TABLE_ROW_STEP } from './flow-summary.consts'
import type { FlowSummaryRef } from './flow-summary.types'

/** Place `specs` as a column at `x`, stepping by `rowStep` and centered vertically around y=0 */
const layoutColumn = (specs: FlowNodeSpec[], x: number, rowStep: number): Node[] => {
    const span = (specs.length - 1) * rowStep
    return specs.map(({ id, type, data }, i) => ({
        id,
        type,
        position: { x, y: i * rowStep - span / 2 },
        data,
    }))
}

/**
 * Turn declarative node-config children (`<FlowSummaryTable>` / `<FlowSummarySink>`) into
 * React Flow nodes, arranging tables into a left column and sinks into a right column in
 * declaration order — callers don't supply coordinates — and caption each column. Other
 * children are returned as-is so they pass through to `<ReactFlow>` as overlays.
 */
export const partitionChildren = (children: ReactNode): { nodes: Node[]; rest: ReactNode[] } => {
    const { specs, rest } = collectNodeSpecs(children)

    const tableNodes = layoutColumn(
        specs.filter((spec) => spec.type === 'table'),
        TABLE_COLUMN_X,
        TABLE_ROW_STEP,
    )
    const sinkNodes = layoutColumn(
        specs.filter((spec) => spec.type === 'sink'),
        SINK_COLUMN_X,
        SINK_ROW_STEP,
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
 * `animation-delay`, and wires the imperative `animate(tableId, sinkId, durationMs?)` handle
 * onto `ref`.
 */
export const useSummaryEdgeAnimation = (
    edges: Edge[] | undefined,
    ref: Ref<FlowSummaryRef> | undefined,
): ReadonlyMap<string, number> => {
    const { animatedEdges, animateEdge } = useEdgeAnimation()

    const edgesRef = useRef(edges)
    edgesRef.current = edges

    const animate = useCallback<FlowSummaryRef['animate']>(
        (tableId, sinkId, durationMs) => {
            const edge = edgesRef.current?.find((e) => e.source === tableId && e.target === sinkId)
            if (edge) {
                animateEdge(edge.id, durationMs)
            }
        },
        [animateEdge],
    )

    useImperativeHandle(ref, () => ({ animate }), [animate])

    return animatedEdges
}
