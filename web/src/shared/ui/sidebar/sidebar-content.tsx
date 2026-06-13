import type { ComponentProps, FC } from 'react'

import { cn } from '@/shared/utils'

type SidebarContentProps = ComponentProps<'div'>

export const SidebarContent: FC<SidebarContentProps> = ({ className, ...props }) => (
    <div
        data-slot="sidebar-content"
        data-sidebar="content"
        className={cn(
            'flex min-h-0 flex-1 flex-col gap-2 overflow-auto group-data-[collapsible=icon]:overflow-hidden',
            className,
        )}
        {...props}
    />
)
