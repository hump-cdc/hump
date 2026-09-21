import type { FC, ReactNode } from 'react'

import { useTopologyModeStore } from '@/entities/flows/toolbar/@x/flows'

import { useFlowsTopology } from '../model/use-flows-topology'
import { FlowsDetailed } from './flows.detailed'
import { FlowsSummary } from './flows.summary'

type FlowsProps = {
    /** Rendered inside the canvas region (the positioned ancestor for floating overlays like the inspector). */
    overlay?: ReactNode
}

/**
 * Flows page topology: a full-bleed diagram wiring source tables (left) to sinks (right),
 * rendered from the metrics endpoint as the summary or detailed view per the mode switched
 * from the toolbar. The page supplies the `overlay` (the inspector drawer) so it docks to the
 * corner of the canvas while staying interactive.
 */
export const Flows: FC<FlowsProps> = ({ overlay }) => {
    const mode = useTopologyModeStore((state) => state.mode)
    const topology = useFlowsTopology()

    return (
        <div className="relative min-h-0 flex-1 overflow-hidden">
            {mode === 'detailed' ? <FlowsDetailed topology={topology} /> : <FlowsSummary topology={topology} />}
            {overlay}
        </div>
    )
}
