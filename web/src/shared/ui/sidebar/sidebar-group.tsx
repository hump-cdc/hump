import type { ComponentProps, FC } from 'react'

import { cn } from '@/shared/utils'

type SidebarGroupProps = ComponentProps<'div'>

export const SidebarGroup: FC<SidebarGroupProps> = ({ className, ...props }) => (
    <div
        data-slot="sidebar-group"
        data-sidebar="group"
        className={cn('relative flex w-full min-w-0 flex-col p-3', className)}
        {...props}
    />
)
