import type { ComponentProps, FC } from 'react'

import { Portal as DropdownMenuPortalPrimitive } from '@radix-ui/react-dropdown-menu'

type DropdownMenuPortalProps = ComponentProps<typeof DropdownMenuPortalPrimitive>

export const DropdownMenuPortal: FC<DropdownMenuPortalProps> = (props) => (
    <DropdownMenuPortalPrimitive data-slot="dropdown-menu-portal" {...props} />
)
