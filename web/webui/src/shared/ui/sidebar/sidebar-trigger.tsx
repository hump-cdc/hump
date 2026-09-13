import type { ComponentProps, FC } from 'react'

import { Menu } from 'lucide-react'

import { cn } from '@/shared/utils'

import { useSidebar } from './sidebar-context'

type SidebarTriggerProps = ComponentProps<'button'>

export const SidebarTrigger: FC<SidebarTriggerProps> = ({ className, onClick, children, ...props }) => {
    const { toggleSidebar } = useSidebar()

    return (
        <button
            type="button"
            data-sidebar="trigger"
            data-slot="sidebar-trigger"
            aria-label="Toggle sidebar"
            className={cn(
                'inline-flex size-8 shrink-0 cursor-pointer items-center justify-center rounded-lg text-ink-subtle transition-colors hover:bg-surface-2 hover:text-ink focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary [&>svg]:size-4.5',
                className,
            )}
            onClick={(event) => {
                onClick?.(event)
                toggleSidebar()
            }}
            {...props}
        >
            {children ?? <Menu />}
            <span className="sr-only">Toggle sidebar</span>
        </button>
    )
}
