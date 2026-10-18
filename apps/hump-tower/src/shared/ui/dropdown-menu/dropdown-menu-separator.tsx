import type { ComponentProps, FC } from 'react'

import { Separator as DropdownMenuSeparatorPrimitive } from '@radix-ui/react-dropdown-menu'

import { cn } from '@/shared/utils'

type DropdownMenuSeparatorProps = ComponentProps<typeof DropdownMenuSeparatorPrimitive>

export const DropdownMenuSeparator: FC<DropdownMenuSeparatorProps> = ({ className, ...props }) => (
    <DropdownMenuSeparatorPrimitive
        data-slot="dropdown-menu-separator"
        className={cn('-mx-1 my-1 h-px bg-hairline', className)}
        {...props}
    />
)
