import type { ComponentProps, FC } from 'react'

import { cn } from '@/shared/utils'

type SidebarSeparatorProps = ComponentProps<'div'>

export const SidebarSeparator: FC<SidebarSeparatorProps> = ({ className, ...props }) => (
    <div
        data-slot="sidebar-separator"
        data-sidebar="separator"
        className={cn('mx-2 my-1 h-px shrink-0 bg-hairline', className)}
        {...props}
    />
)
