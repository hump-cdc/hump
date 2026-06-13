import type { FC } from 'react'

import { GitCompareArrows, X } from 'lucide-react'

import { SheetClose, SheetHeader, SheetTitle } from '@/shared/ui/sheet'

/** Static inspector header: icon, title and the sheet close button. */
export const InspectorHeader: FC = () => (
    <SheetHeader className="flex-row items-center justify-between gap-2 border-b border-hairline p-4">
        <div className="flex items-center gap-2">
            <GitCompareArrows className="size-4 text-ink-subtle" />
            <SheetTitle className="text-base">Inspector</SheetTitle>
        </div>
        <SheetClose aria-label="Close inspector" className="flex text-ink-tertiary transition-colors hover:text-ink">
            <X className="size-4" />
        </SheetClose>
    </SheetHeader>
)
