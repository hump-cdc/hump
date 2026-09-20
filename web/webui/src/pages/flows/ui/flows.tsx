import type { FC } from 'react'

import { Flows as FlowsCanvas } from '@/entities/flows/flows'
import { FlowInspector, useFlowInspectorStore } from '@/entities/flows/inspector'
import { FlowsToolbar } from '@/entities/flows/toolbar'
import { cn, useIsMobile } from '@/shared/utils'

export const Flows: FC = () => {
    const isInspectorOpen = useFlowInspectorStore((state) => state.selected !== null)
    const isMobile = useIsMobile()

    // On desktop the floating toolbar yields the inspector's width (300px), in step with the
    // sheet's open (300ms) and close (200ms) animations. On mobile the inspector opens from the bottom.
    const isShrunk = isInspectorOpen && !isMobile

    return (
        <div className="flex min-h-0 flex-1 flex-col">
            <FlowsCanvas
                overlay={
                    <>
                        {/* Floats over the canvas; only the panel itself captures pointer events. */}
                        <div
                            className={cn(
                                'pointer-events-none absolute inset-x-0 top-0 z-10 p-6 transition-[padding] ease-in-out *:pointer-events-auto',
                                isShrunk ? 'pr-81 duration-300' : 'duration-200',
                            )}
                        >
                            <FlowsToolbar />
                        </div>
                        <FlowInspector />
                    </>
                }
            />
        </div>
    )
}
