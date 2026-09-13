import type { ComponentProps, FC } from 'react'

import { Group as DropdownMenuGroupPrimitive } from '@radix-ui/react-dropdown-menu'

type DropdownMenuGroupProps = ComponentProps<typeof DropdownMenuGroupPrimitive>

export const DropdownMenuGroup: FC<DropdownMenuGroupProps> = (props) => (
    <DropdownMenuGroupPrimitive data-slot="dropdown-menu-group" {...props} />
)
