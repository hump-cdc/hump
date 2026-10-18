import { type FC, type ReactNode, useMemo, useRef } from 'react'

import { useFlowInspectorStore } from '@/entities/flows/inspector/@x/flows'
import { useLiveOperations } from '@/shared/live'
import type { ViewportInsets } from '@/shared/ui/flow'
import {
    FlowDetailed,
    type FlowDetailedRef,
    FlowDetailedSink,
    FlowDetailedTable,
    sinkTargetKey,
} from '@/shared/ui/flow-detailed'

import { type FlowsTopology, sinkId, tableId } from '../model/topology'
import type { DetailedSinkSpec } from '../model/types'

/**
 * Detailed topology: sinks broken down into their targets. Clicking a node, target row or edge
 * stores the selection in the shared inspector store, which highlights it. Every live delivery
 * flashes its table → sink target edge.
 */
export const FlowsDetailed: FC<{ topology: FlowsTopology; viewportInsets?: ViewportInsets; controls?: ReactNode }> = ({
    topology: { tables, detailedSinks, detailedEdges },
    viewportInsets,
    controls,
}) => {
    const selected = useFlowInspectorStore((state) => state.selected)
    const selectTable = useFlowInspectorStore((state) => state.selectTable)
    const selectSink = useFlowInspectorStore((state) => state.selectSink)
    const selectTarget = useFlowInspectorStore((state) => state.selectTarget)
    const selectEdge = useFlowInspectorStore((state) => state.selectEdge)
    const clear = useFlowInspectorStore((state) => state.clear)

    const flowRef = useRef<FlowDetailedRef>(null)
    useLiveOperations((batch) => {
        for (const operation of batch) {
            flowRef.current?.animate(
                tableId(operation.schema, operation.table),
                sinkId(operation.sink, operation.host),
                operation.target,
            )
        }
    })

    const labelById = useMemo(
        () => new Map([...tables, ...detailedSinks].map((node) => [node.id, node.name])),
        [tables, detailedSinks],
    )
    const sinkById = useMemo(() => new Map(detailedSinks.map((sink) => [sink.id, sink])), [detailedSinks])

    return (
        <FlowDetailed
            ref={flowRef}
            viewportInsets={viewportInsets}
            controls={controls}
            edges={detailedEdges}
            selected={selected?.id}
            onTableNodeSelect={(id, { name }) => selectTable({ id, name })}
            onSinkNodeSelect={(id, { name, type, host }) => selectSink({ id, name, type, host: host ?? '' })}
            onSinkTargetSelect={(id, target) =>
                selectTarget({
                    id: sinkTargetKey(id, target.id),
                    name: target.name,
                    sinkLabel: labelById.get(id) ?? id,
                    type: sinkById.get(id)?.type ?? '',
                    host: sinkById.get(id)?.host ?? '',
                })
            }
            onEdgeSelect={(id, { source, target, targetHandle }) =>
                selectEdge({
                    id,
                    sourceLabel: labelById.get(source) ?? source,
                    targetLabel: getTargetLabel(detailedSinks, labelById.get(target) ?? target, target, targetHandle),
                    table: source,
                    type: sinkById.get(target)?.type ?? '',
                    host: sinkById.get(target)?.host ?? '',
                    target: targetHandle ?? undefined,
                })
            }
            onClearSelection={clear}
        >
            {tables.map((table) => (
                <FlowDetailedTable key={table.id} {...table} />
            ))}
            {detailedSinks.map((sink) => (
                <FlowDetailedSink key={sink.id} {...sink} />
            ))}
        </FlowDetailed>
    )
}

/** Edge endpoint label: `<sink>/<target>` when the edge lands on a sink target, else the node label. */
const getTargetLabel = (
    sinks: DetailedSinkSpec[],
    nodeLabel: string,
    nodeId: string,
    targetHandle?: string | null,
): string => {
    const target = sinks.find((sink) => sink.id === nodeId)?.targets.find((candidate) => candidate.id === targetHandle)

    return target ? `${nodeLabel}/${target.name}` : nodeLabel
}
