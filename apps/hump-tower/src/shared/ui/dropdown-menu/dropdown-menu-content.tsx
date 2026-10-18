import type { ComponentProps, FC } from 'react'

import { Content as DropdownMenuContentPrimitive, Portal as DropdownMenuPortal } from '@radix-ui/react-dropdown-menu'

import { cn } from '@/shared/utils'

type DropdownMenuContentProps = ComponentProps<typeof DropdownMenuContentPrimitive>

export const DropdownMenuContent: FC<DropdownMenuContentProps> = ({ className, sideOffset = 4, ...props }) => (
    <DropdownMenuPortal>
        <DropdownMenuContentPrimitive
            data-slot="dropdown-menu-content"
            sideOffset={sideOffset}
            className={cn(
                'z-50 max-h-[var(--radix-dropdown-menu-content-available-height)] min-w-32 overflow-y-auto overflow-x-hidden rounded-lg border border-hairline bg-canvas p-1 text-ink shadow-lg',
                'fade-in-0 zoom-in-95 data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95 animate-in data-[state=closed]:animate-out',
                'data-[side=bottom]:slide-in-from-top-1 data-[side=left]:slide-in-from-right-1 data-[side=right]:slide-in-from-left-1 data-[side=top]:slide-in-from-bottom-1',
                className,
            )}
            {...props}
        />
    </DropdownMenuPortal>
)
