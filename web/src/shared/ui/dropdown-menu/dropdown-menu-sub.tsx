import type { ComponentProps, FC } from 'react'

import { Sub as DropdownMenuSubPrimitive } from '@radix-ui/react-dropdown-menu'

type DropdownMenuSubProps = ComponentProps<typeof DropdownMenuSubPrimitive>

export const DropdownMenuSub: FC<DropdownMenuSubProps> = (props) => (
    <DropdownMenuSubPrimitive data-slot="dropdown-menu-sub" {...props} />
)
