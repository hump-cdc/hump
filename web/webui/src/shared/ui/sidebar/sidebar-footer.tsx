import type { ComponentProps, FC } from 'react'

import { cn } from '@/shared/utils'

type SidebarFooterProps = ComponentProps<'div'>

export const SidebarFooter: FC<SidebarFooterProps> = ({ className, ...props }) => (
    <div
        data-slot="sidebar-footer"
        data-sidebar="footer"
        className={cn('flex flex-col gap-1 p-3', className)}
        {...props}
    />
)
