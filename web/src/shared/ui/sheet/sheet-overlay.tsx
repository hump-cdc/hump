import type { ComponentProps, FC } from 'react'

import { Overlay as SheetOverlayPrimitive } from '@radix-ui/react-dialog'

import { cn } from '@/shared/utils'

type SheetOverlayProps = ComponentProps<typeof SheetOverlayPrimitive>

export const SheetOverlay: FC<SheetOverlayProps> = ({ className, ...props }) => (
    <SheetOverlayPrimitive
        data-slot="sheet-overlay"
        className={cn(
            'fixed inset-0 z-50 bg-black/50 fade-in-0 animate-in data-[state=closed]:fade-out-0 data-[state=closed]:animate-out',
            className,
        )}
        {...props}
    />
)
