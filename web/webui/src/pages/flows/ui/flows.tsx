import type { FC } from 'react'

import { Flows as FlowsCanvas } from '@/entities/flows/flows'
import { FlowInspector } from '@/entities/flows/inspector'
import { FlowsToolbar } from '@/entities/flows/toolbar'
import { cn } from '@/shared/utils'

import { useInsets } from './use-insets'

export const Flows: FC = () => {
    const { toolbarRef, inspectorRef, isShrunk, viewportInsets } = useInsets()

    return (
        <div className="flex min-h-0 flex-1 flex-col">
            <FlowsCanvas
                viewportInsets={viewportInsets}
                overlay={
                    <>
                        {/* Floats over the canvas; only the panel itself captures pointer events. */}
                        <div
                            ref={toolbarRef}
                            className={cn(
                                'pointer-events-none absolute inset-x-0 top-0 z-10 p-6 transition-[padding] ease-in-out *:pointer-events-auto',
                                isShrunk ? 'pr-81 duration-300' : 'duration-200',
                            )}
                        >
                            <FlowsToolbar />
                        </div>
                        <FlowInspector ref={inspectorRef} />
                    </>
                }
            />
        </div>
    )
}
