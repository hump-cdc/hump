import type { ComponentProps, FC } from 'react'

import { Root as DropdownMenuRoot } from '@radix-ui/react-dropdown-menu'

type DropdownMenuProps = ComponentProps<typeof DropdownMenuRoot>

export const DropdownMenu: FC<DropdownMenuProps> = (props) => <DropdownMenuRoot data-slot="dropdown-menu" {...props} />
