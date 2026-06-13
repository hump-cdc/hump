import type { ComponentProps, FC } from 'react'

import { cn } from '@/shared/utils'

type SidebarMenuItemProps = ComponentProps<'li'>

export const SidebarMenuItem: FC<SidebarMenuItemProps> = ({ className, ...props }) => (
    <li
        data-slot="sidebar-menu-item"
        data-sidebar="menu-item"
        className={cn('group/menu-item relative', className)}
        {...props}
    />
)
