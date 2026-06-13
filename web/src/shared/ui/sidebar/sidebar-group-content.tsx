import type { ComponentProps, FC } from 'react'

import { cn } from '@/shared/utils'

type SidebarGroupContentProps = ComponentProps<'div'>

export const SidebarGroupContent: FC<SidebarGroupContentProps> = ({ className, ...props }) => (
    <div
        data-slot="sidebar-group-content"
        data-sidebar="group-content"
        className={cn('w-full text-sm', className)}
        {...props}
    />
)
