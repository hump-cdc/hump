import { type FC, type ReactNode, useMemo, useRef } from 'react'

import { useFlowInspectorStore } from '@/entities/flows/inspector/@x/flows'
import { useLiveOperations } from '@/shared/live'
import type { ViewportInsets } from '@/shared/ui/flow'
import { FlowSummary, type FlowSummaryRef, FlowSummarySink, FlowSummaryTable } from '@/shared/ui/flow-summary'

import { type FlowsTopology, sinkId, tableId } from '../model/topology'

/**
 * Summary topology: one card per sink. Clicking a node or edge stores the selection in the
 * shared inspector store, which highlights it. Every live delivery flashes its table → sink edge.
 */
export const FlowsSummary: FC<{ topology: FlowsTopology; viewportInsets?: ViewportInsets; controls?: ReactNode }> = ({
    topology: { tables, sinks, edges },
    viewportInsets,
    controls,
}) => {
    const selected = useFlowInspectorStore((state) => state.selected)
    const selectTable = useFlowInspectorStore((state) => state.selectTable)
    const selectSink = useFlowInspectorStore((state) => state.selectSink)
    const selectEdge = useFlowInspectorStore((state) => state.selectEdge)
    const clear = useFlowInspectorStore((state) => state.clear)

    const flowRef = useRef<FlowSummaryRef>(null)
    useLiveOperations((batch) => {
        for (const operation of batch) {
            flowRef.current?.animate(tableId(operation.schema, operation.table), sinkId(operation.sink, operation.host))
        }
    })

    const labelById = useMemo(() => new Map([...tables, ...sinks].map((node) => [node.id, node.name])), [tables, sinks])

    return (
        <FlowSummary
            ref={flowRef}
            viewportInsets={viewportInsets}
            controls={controls}
            edges={edges}
            selected={selected?.id}
            onTableNodeSelect={selectTable}
            onSinkNodeSelect={selectSink}
            onEdgeSelect={(id, { source, target }) =>
                selectEdge({
                    id,
                    sourceLabel: labelById.get(source) ?? source,
                    targetLabel: labelById.get(target) ?? target,
                })
            }
            onClearSelection={clear}
        >
            {tables.map((table) => (
                <FlowSummaryTable key={table.id} {...table} />
            ))}
            {sinks.map((sink) => (
                <FlowSummarySink key={sink.id} {...sink} />
            ))}
        </FlowSummary>
    )
}
