import { type FC, useEffect, useState } from 'react'

import { ArrowRight, ReceiptText, X } from 'lucide-react'

import { Sheet, SheetClose, SheetContent, SheetHeader, SheetTitle } from '@/shared/ui/sheet'
import { cn, useIsMobile } from '@/shared/utils'

import { useOperationDetailsStore } from '../model/store'
import { OperationDetailsPanel } from './operation-details.panel'
import { OperationDetailsTabs } from './operation-details.tabs'

/**
 * Non-blocking details drawer for the operations list, built on the shared `Sheet`. Docks to the
 * right edge of the grid region at full grid height (viewport bottom on mobile) and stays modal-less
 * so the rows beneath remain interactive. Opens while an operation is selected.
 */
export const OperationDetails: FC = () => {
    const selected = useOperationDetailsStore((state) => state.selected)
    const clear = useOperationDetailsStore((state) => state.clear)
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
                // On mobile, portal to the viewport so the bottom sheet spans edge to edge,
                // outside the page padding around the grid.
                portal={isMobile}
                showOverlay={false}
                showCloseButton={false}
                aria-describedby={undefined}
                onOpenAutoFocus={(event) => event.preventDefault()}
                onInteractOutside={(event) => event.preventDefault()}
                onPointerDownOutside={(event) => event.preventDefault()}
                className={cn(
                    'gap-0 bg-surface-1 p-0',
                    isMobile ? 'h-[80%] rounded-t-xl' : 'absolute z-10 h-full w-95 max-w-none rounded-xl border',
                )}
            >
                {isMobile ? (
                    <SheetHeader className="flex-row items-center justify-between gap-2 border-transparent border-b p-4">
                        <SheetTitle className="flex min-w-0 items-center gap-2 font-medium font-mono text-md">
                            <span className="truncate">{shown?.table}</span>
                            <ArrowRight className="size-3.75 shrink-0 text-ink-tertiary" />
                            <span className="truncate">{shown?.sink}</span>
                        </SheetTitle>
                        <CloseButton />
                    </SheetHeader>
                ) : (
                    <SheetHeader className="flex-row items-center justify-between gap-2 border-hairline border-b px-4 py-3.5">
                        <div className="flex items-center gap-2">
                            <ReceiptText className="size-3.75 text-ink-subtle" />
                            <SheetTitle className="text-md">Operation details</SheetTitle>
                        </div>
                        <CloseButton />
                    </SheetHeader>
                )}

                {shown &&
                    (isMobile ? (
                        <OperationDetailsTabs key={shown.id} operation={shown} />
                    ) : (
                        <OperationDetailsPanel operation={shown} />
                    ))}
            </SheetContent>
        </Sheet>
    )
}

const CloseButton: FC = () => (
    <SheetClose aria-label="Close details" className="flex text-ink-tertiary transition-colors hover:text-ink">
        <X className="size-4" />
    </SheetClose>
)
