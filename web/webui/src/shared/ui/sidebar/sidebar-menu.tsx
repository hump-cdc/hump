import type { ComponentProps, FC } from 'react'

import { cn } from '@/shared/utils'

type SidebarMenuProps = ComponentProps<'ul'>

export const SidebarMenu: FC<SidebarMenuProps> = ({ className, ...props }) => (
    <ul
        data-slot="sidebar-menu"
        data-sidebar="menu"
        className={cn('flex w-full min-w-0 flex-col gap-1', className)}
        {...props}
    />
)
