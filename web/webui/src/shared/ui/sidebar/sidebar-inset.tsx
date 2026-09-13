import type { ComponentProps, FC } from 'react'

import { cn } from '@/shared/utils'

type SidebarInsetProps = ComponentProps<'main'>

export const SidebarInset: FC<SidebarInsetProps> = ({ className, ...props }) => (
    <main
        data-slot="sidebar-inset"
        className={cn('relative flex min-h-svh min-w-0 flex-1 flex-col bg-canvas', className)}
        {...props}
    />
)
