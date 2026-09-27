import { type AnimationEvent, type FC, type ReactNode, useMemo } from 'react'

import type { ViewportInsets } from '@/shared/ui/flow'
import { cn, useSwitchAnimation } from '@/shared/utils'

import { filterFlowsTopology, useFlowsFilterStore } from '../model/filter'
import { useTopologyModeStore } from '../model/mode'
import { useFlowsTopology } from '../model/use-flows-topology'
import { FlowsDetailed } from './flows.detailed'
import { FlowsModeSwitch } from './flows.mode'
import { FlowsSummary } from './flows.summary'

type FlowsProps = {
    /** Rendered inside the canvas region (the positioned ancestor for floating overlays like the inspector). */
    overlay?: ReactNode
    /** Canvas margins covered by the `overlay`, kept clear when the diagram pans to a selection. */
    viewportInsets?: ViewportInsets
}

/**
 * Flows page topology: a full-bleed diagram wiring source tables (left) to sinks (right),
 * rendered from the metrics endpoint (narrowed to the tables and sinks picked in the toolbar) as the summary or detailed view per the mode switched
 * from the canvas zoom controls; switching modes cross-fades the views. The page supplies the `overlay` (the inspector drawer) so it docks to the
 * corner of the canvas while staying interactive.
 */
export const Flows: FC<FlowsProps> = ({ overlay, viewportInsets }) => {
    const mode = useTopologyModeStore((state) => state.mode)
    const fullTopology = useFlowsTopology()
    const tables = useFlowsFilterStore((state) => state.tables)
    const sinks = useFlowsFilterStore((state) => state.sinks)
    const topology = useMemo(() => filterFlowsTopology(fullTopology, tables, sinks), [fullTopology, tables, sinks])

    // The view being switched away from stays mounted until its fade-out ends.
    const { current, leaving, views, onLeaveEnd } = useSwitchAnimation(mode)

    return (
        <div className="relative min-h-0 flex-1 overflow-hidden">
            {views.map((view) => {
                const isLeaving = view !== current

                return (
                    <div
                        key={view}
                        inert={isLeaving}
                        className={cn(
                            'absolute inset-0 duration-500 ease-in-out',
                            // The first view appears without a fade; afterwards the incoming one fades in over the outgoing one.
                            leaving !== null &&
                                (isLeaving ? 'fade-out-0 animate-out fill-mode-forwards' : 'fade-in-0 animate-in'),
                        )}
                        // Only the layer's own fade ends the transition, not animations bubbling from the diagram.
                        onAnimationEnd={(event: AnimationEvent) => {
                            if (isLeaving && event.target === event.currentTarget) {
                                onLeaveEnd()
                            }
                        }}
                    >
                        {view === 'detailed' ? (
                            <FlowsDetailed
                                topology={topology}
                                viewportInsets={viewportInsets}
                                controls={<FlowsModeSwitch />}
                            />
                        ) : (
                            <FlowsSummary
                                topology={topology}
                                viewportInsets={viewportInsets}
                                controls={<FlowsModeSwitch />}
                            />
                        )}
                    </div>
                )
            })}
            {overlay}
        </div>
    )
}
