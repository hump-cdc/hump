import type { ComponentProps, FC } from 'react'

import { cn } from '@/shared/utils'

type SidebarHeaderProps = ComponentProps<'div'>

export const SidebarHeader: FC<SidebarHeaderProps> = ({ className, ...props }) => (
    <div
        data-slot="sidebar-header"
        data-sidebar="header"
        className={cn('flex select-none flex-col gap-2 p-3', className)}
        {...props}
    />
)
