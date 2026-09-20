import { type FC, type ReactNode, useMemo } from 'react'

import { useFlowInspectorStore } from '@/entities/flows/inspector/@x/topology'
import { useLiveOperations } from '@/shared/live'
import { Flow, SinkNode, TableNode } from '@/shared/ui/flow'

import { edges, sinks, tables } from '../model/mock'
import { TopologyToolbar } from './topology-toolbar'

type FlowsTopologyProps = {
    /** Rendered inside the canvas region (the positioned ancestor for floating overlays like the inspector). */
    overlay?: ReactNode
}

/**
 * Flows page topology: the toolbar above a full-bleed `Flow` diagram wiring source
 * tables (left) to sinks (right). Clicking a node or edge stores the selection in the
 * shared inspector store, which highlights it. The page supplies the `overlay` (the
 * inspector drawer) so it docks to the corner of the canvas while staying interactive.
 */
export const FlowsTopology: FC<FlowsTopologyProps> = ({ overlay }) => {
    const selected = useFlowInspectorStore((state) => state.selected)
    const selectTable = useFlowInspectorStore((state) => state.selectTable)
    const selectSink = useFlowInspectorStore((state) => state.selectSink)
    const selectEdge = useFlowInspectorStore((state) => state.selectEdge)
    const clear = useFlowInspectorStore((state) => state.clear)

    const labelById = useMemo(() => new Map([...tables, ...sinks].map((node) => [node.id, node.name])), [])

    // Placeholder consumer: this is where the live feed will animate the matching edges.
    useLiveOperations((batch) => console.log('live batch', batch))

    return (
        <div className="flex min-h-0 flex-1 flex-col">
            <TopologyToolbar />
            <div className="relative min-h-0 flex-1 overflow-hidden">
                <Flow
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
                        <TableNode key={table.id} {...table} />
                    ))}
                    {sinks.map((sink) => (
                        <SinkNode key={sink.id} {...sink} />
                    ))}
                </Flow>
                {overlay}
            </div>
        </div>
    )
}
