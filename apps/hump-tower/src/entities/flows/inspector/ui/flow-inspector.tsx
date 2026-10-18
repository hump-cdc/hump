import { Link } from '@tanstack/react-router'

import { type FC, type Ref, useEffect, useState } from 'react'

import { ArrowUpRight, type LucideIcon, Shuffle, Table2, Workflow, X } from 'lucide-react'

import { Button } from '@/shared/ui/button'
import { Sheet, SheetClose, SheetContent, SheetHeader, SheetTitle } from '@/shared/ui/sheet'
import { cn, useIsMobile } from '@/shared/utils'

import { toOperationsSearch } from '../model/operations-search'
import { useFlowInspectorStore } from '../model/store'
import type { SelectedItem } from '../model/types'
import { InspectorDetails } from './inspector-details'
import { RecentOperations } from './recent-operations'

/** Header title and icon per selection kind; a target is a part of a sink, so it reads as one. */
const HEADERS: Record<SelectedItem['kind'], { title: string; icon: LucideIcon }> = {
    table: { title: 'Table details', icon: Table2 },
    sink: { title: 'Sink details', icon: Workflow },
    target: { title: 'Sink details', icon: Workflow },
    edge: { title: 'Connection details', icon: Shuffle },
}

/**
 * Non-blocking inspector for the flows topology, built on the shared `Sheet`. Docks to the
 * right edge of the canvas region at full canvas height and renders the selected node/edge
 * details (mirrors the `Inspector` design frame). Driven by the shared store: opens while a
 * selection exists and stays modal-less so the canvas beneath remains interactive. `ref` points at
 * the sheet panel, e.g. to measure the canvas area it covers.
 */
export const FlowInspector: FC<{ ref?: Ref<HTMLDivElement> }> = ({ ref }) => {
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

    const header = shown ? HEADERS[shown.kind] : null

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
                ref={ref}
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
                <SheetHeader className="flex-row items-center justify-between gap-2 border-hairline border-b p-4">
                    <div className="flex items-center gap-2">
                        {header ? <header.icon className="size-4 text-ink-subtle" /> : null}
                        <SheetTitle className="text-base">{header?.title}</SheetTitle>
                    </div>
                    <SheetClose
                        aria-label="Close inspector"
                        className="flex text-ink-tertiary transition-colors hover:text-ink"
                    >
                        <X className="size-4" />
                    </SheetClose>
                </SheetHeader>

                {shown ? (
                    <div className="flex flex-1 flex-col gap-4 overflow-y-auto p-4">
                        <InspectorDetails selected={shown} />

                        {!isMobile && <RecentOperations selected={shown} />}

                        <Button className="mt-auto" asChild>
                            <Link to="/operations" search={toOperationsSearch(shown)}>
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
