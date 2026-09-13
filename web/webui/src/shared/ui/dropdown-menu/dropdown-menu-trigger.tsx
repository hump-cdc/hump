import type { ComponentProps, FC } from 'react'

import { Trigger as DropdownMenuTriggerPrimitive } from '@radix-ui/react-dropdown-menu'

type DropdownMenuTriggerProps = ComponentProps<typeof DropdownMenuTriggerPrimitive>

export const DropdownMenuTrigger: FC<DropdownMenuTriggerProps> = (props) => (
    <DropdownMenuTriggerPrimitive data-slot="dropdown-menu-trigger" {...props} />
)
