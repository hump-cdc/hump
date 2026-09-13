import type { ComponentProps, FC } from 'react'

import { SubContent as DropdownMenuSubContentPrimitive } from '@radix-ui/react-dropdown-menu'

import { cn } from '@/shared/utils'

type DropdownMenuSubContentProps = ComponentProps<typeof DropdownMenuSubContentPrimitive>

export const DropdownMenuSubContent: FC<DropdownMenuSubContentProps> = ({ className, ...props }) => (
    <DropdownMenuSubContentPrimitive
        data-slot="dropdown-menu-sub-content"
        className={cn(
            'z-50 min-w-32 overflow-hidden rounded-lg border border-hairline bg-canvas p-1 text-ink shadow-lg',
            'fade-in-0 zoom-in-95 animate-in data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95 data-[state=closed]:animate-out',
            'data-[side=bottom]:slide-in-from-top-1 data-[side=left]:slide-in-from-right-1 data-[side=right]:slide-in-from-left-1 data-[side=top]:slide-in-from-bottom-1',
            className,
        )}
        {...props}
    />
)
