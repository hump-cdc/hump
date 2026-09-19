import type { ComponentProps, FC } from 'react'

import { Overlay as SheetOverlayPrimitive } from '@radix-ui/react-dialog'

import { cn } from '@/shared/utils'

type SheetOverlayProps = ComponentProps<typeof SheetOverlayPrimitive>

export const SheetOverlay: FC<SheetOverlayProps> = ({ className, ...props }) => (
    <SheetOverlayPrimitive
        data-slot="sheet-overlay"
        className={cn(
            'fade-in-0 data-[state=closed]:fade-out-0 fixed inset-0 z-50 animate-in bg-black/50 data-[state=closed]:animate-out',
            className,
        )}
        {...props}
    />
)
