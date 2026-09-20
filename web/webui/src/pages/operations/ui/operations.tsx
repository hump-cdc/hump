import type { FC } from 'react'

import { OperationDetails, useOperationDetailsStore } from '@/entities/operations/details'
import { OperationsFilter } from '@/entities/operations/filter'
import { OperationsList } from '@/entities/operations/list'
import { Toolbar } from '@/shared/ui/toolbar'
import { cn, useIsMobile } from '@/shared/utils'

export const Operations: FC = () => {
    const isDetailsOpen = useOperationDetailsStore((state) => state.selected !== null)
    const isMobile = useIsMobile()

    // On desktop the grid yields the drawer's width (380px) plus a gap, in step with the sheet's
    // open (300ms) and close (200ms) animations. On mobile the drawer overlays from the bottom.
    const isShrunk = isDetailsOpen && !isMobile

    return (
        <div className="flex flex-1 flex-col gap-4 p-6">
            <Toolbar>
                <OperationsFilter />
            </Toolbar>
            {/* Positioned, clipped grid region: the details drawer docks inside it at the grid's height. */}
            <div
                className={cn(
                    'relative flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl transition-[padding] ease-in-out',
                    isShrunk ? 'pr-99 duration-300' : 'duration-200',
                )}
            >
                <OperationsList />
                <OperationDetails />
            </div>
        </div>
    )
}
