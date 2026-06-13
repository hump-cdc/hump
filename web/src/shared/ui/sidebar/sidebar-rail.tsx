import type { ComponentProps, FC } from 'react'

import { cn } from '@/shared/utils'

import { useSidebar } from './sidebar-context'

type SidebarRailProps = ComponentProps<'button'>

export const SidebarRail: FC<SidebarRailProps> = ({ className, ...props }) => {
    const { toggleSidebar } = useSidebar()

    return (
        <button
            type="button"
            data-sidebar="rail"
            data-slot="sidebar-rail"
            aria-label="Toggle sidebar"
            tabIndex={-1}
            title="Toggle sidebar"
            onClick={toggleSidebar}
            className={cn(
                'absolute inset-y-0 z-20 hidden w-4 -translate-x-1/2 transition-all ease-linear lg:flex',
                'after:absolute after:inset-y-0 after:left-1/2 after:w-px hover:after:bg-hairline-strong',
                'group-data-[side=left]:-right-4 group-data-[side=right]:left-0',
                'cursor-ew-resize',
                className,
            )}
            {...props}
        />
    )
}
