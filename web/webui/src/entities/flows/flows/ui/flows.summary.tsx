import { type FC, useMemo } from 'react'

import { useFlowInspectorStore } from '@/entities/flows/inspector/@x/flows'
import { FlowSummary, FlowSummarySink, FlowSummaryTable } from '@/shared/ui/flow-summary'

import { edges, sinks, tables } from '../model/mock'

/**
 * Summary topology: one card per sink. Clicking a node or edge stores the selection in the
 * shared inspector store, which highlights it.
 */
export const FlowsSummary: FC = () => {
    const selected = useFlowInspectorStore((state) => state.selected)
    const selectTable = useFlowInspectorStore((state) => state.selectTable)
    const selectSink = useFlowInspectorStore((state) => state.selectSink)
    const selectEdge = useFlowInspectorStore((state) => state.selectEdge)
    const clear = useFlowInspectorStore((state) => state.clear)

    const labelById = useMemo(() => new Map([...tables, ...sinks].map((node) => [node.id, node.name])), [])

    return (
        <FlowSummary
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
