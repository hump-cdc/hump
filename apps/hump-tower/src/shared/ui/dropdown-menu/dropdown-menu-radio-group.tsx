import type { ComponentProps, FC } from 'react'

import { RadioGroup as DropdownMenuRadioGroupPrimitive } from '@radix-ui/react-dropdown-menu'

type DropdownMenuRadioGroupProps = ComponentProps<typeof DropdownMenuRadioGroupPrimitive>

export const DropdownMenuRadioGroup: FC<DropdownMenuRadioGroupProps> = (props) => (
    <DropdownMenuRadioGroupPrimitive data-slot="dropdown-menu-radio-group" {...props} />
)
