import { Link } from '@tanstack/react-router'

import { type FC, useEffect, useState } from 'react'

import { ArrowUpRight } from 'lucide-react'

import { Button } from '@/shared/ui/button'
import { Sheet, SheetContent } from '@/shared/ui/sheet'
import { cn, useIsMobile } from '@/shared/utils'

import { useFlowInspectorStore } from '../model/store'
import { InspectorDetails } from './inspector-details'
import { InspectorHeader } from './inspector-header'
import { RecentOperations } from './recent-operations'

/**
 * Non-blocking inspector for the flows topology, built on the shared `Sheet`. Docks to the
 * right edge of the canvas region at full canvas height and renders the selected node/edge
 * details (mirrors the `Inspector` design frame). Driven by the shared store: opens while a
 * selection exists and stays modal-less so the canvas beneath remains interactive.
 */
export const FlowInspector: FC = () => {
    const selected = useFlowInspectorStore((state) => state.selected)
    const clear = useFlowInspectorStore((state) => state.clear)
    const isMobile = useIsMobile()

    // Keep rendering the last selection through the sheet's close animation, after which
    // `selected` is already null.
    const [shown, setShown] = useState(selected)
    useEffect(() => {
        if (selected) {
            setShown(selected)
        }
    }, [selected])

    return (
        <Sheet
            modal={false}
            open={Boolean(selected)}
            onOpenChange={(open) => {
                if (!open) {
                    clear()
                }
            }}
        >
            <SheetContent
                side={isMobile ? 'bottom' : 'right'}
                portal={false}
                showOverlay={false}
                showCloseButton={false}
                aria-describedby={undefined}
                onOpenAutoFocus={(event) => event.preventDefault()}
                onInteractOutside={(event) => event.preventDefault()}
                onPointerDownOutside={(event) => event.preventDefault()}
                className={cn(
                    'absolute z-10 gap-0 bg-surface-1 p-0',
                    isMobile ? 'max-h-[80%] rounded-t-xl' : 'h-full w-75 max-w-none rounded-none',
                )}
            >
                <InspectorHeader />

                {shown ? (
                    <div className="flex flex-1 flex-col gap-4 overflow-y-auto p-4">
                        <InspectorDetails selected={shown} />

                        {!isMobile && <RecentOperations />}

                        <Button className="mt-auto" asChild>
                            <Link to="/operations">
                                View in Operations
                                <ArrowUpRight />
                            </Link>
                        </Button>
                    </div>
                ) : null}
            </SheetContent>
        </Sheet>
    )
}
