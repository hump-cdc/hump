import type { ComponentProps, FC } from 'react'

import { cn } from '@/shared/utils'

type DropdownMenuShortcutProps = ComponentProps<'span'>

export const DropdownMenuShortcut: FC<DropdownMenuShortcutProps> = ({ className, ...props }) => (
    <span
        data-slot="dropdown-menu-shortcut"
        className={cn('ml-auto text-ink-subtle text-xs tracking-widest', className)}
        {...props}
    />
)
