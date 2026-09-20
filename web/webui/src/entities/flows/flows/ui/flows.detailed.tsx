import { type FC, useMemo } from 'react'

import { useFlowInspectorStore } from '@/entities/flows/inspector/@x/flows'
import { FlowDetailed, FlowDetailedGhost, FlowDetailedSink, FlowDetailedTable } from '@/shared/ui/flow-detailed'

import { detailedEdges, detailedSinks, ghosts, tables } from '../model/mock'

/**
 * Detailed topology: sinks broken down into their targets, plus ghost placeholders for
 * unconfigured sink types. Clicking a node, target row or edge stores the selection in the
 * shared inspector store, which highlights it.
 */
export const FlowsDetailed: FC = () => {
    const selected = useFlowInspectorStore((state) => state.selected)
    const selectTable = useFlowInspectorStore((state) => state.selectTable)
    const selectDetailedSink = useFlowInspectorStore((state) => state.selectDetailedSink)
    const selectTarget = useFlowInspectorStore((state) => state.selectTarget)
    const selectEdge = useFlowInspectorStore((state) => state.selectEdge)
    const clear = useFlowInspectorStore((state) => state.clear)

    const labelById = useMemo(
        () =>
            new Map([
                ...[...tables, ...detailedSinks].map((node) => [node.id, node.name] as const),
                ...ghosts.map((ghost) => [ghost.id, ghost.type] as const),
            ]),
        [],
    )

    return (
        <FlowDetailed
            edges={detailedEdges}
            selected={selected?.id}
            onTableNodeSelect={selectTable}
            onSinkNodeSelect={selectDetailedSink}
            onSinkTargetSelect={(sinkId, target) =>
                selectTarget({ sinkId, sinkLabel: labelById.get(sinkId) ?? sinkId, target })
            }
            onEdgeSelect={(id, { source, target, targetHandle }) =>
                selectEdge({
                    id,
                    sourceLabel: labelById.get(source) ?? source,
                    targetLabel: getTargetLabel(labelById.get(target) ?? target, target, targetHandle),
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
            {ghosts.map((ghost) => (
                <FlowDetailedGhost key={ghost.id} {...ghost} />
            ))}
        </FlowDetailed>
    )
}

/** Edge endpoint label: `<sink>/<target>` when the edge lands on a sink target, else the node label. */
const getTargetLabel = (nodeLabel: string, nodeId: string, targetHandle?: string | null): string => {
    const target = detailedSinks
        .find((sink) => sink.id === nodeId)
        ?.targets.find((candidate) => candidate.id === targetHandle)

    return target ? `${nodeLabel}/${target.name}` : nodeLabel
}
