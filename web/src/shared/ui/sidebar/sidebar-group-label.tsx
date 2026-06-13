import type { ComponentProps, FC } from 'react'

import { Slot } from '@radix-ui/react-slot'

import { cn } from '@/shared/utils'

type SidebarGroupLabelProps = ComponentProps<'div'> & { asChild?: boolean }

export const SidebarGroupLabel: FC<SidebarGroupLabelProps> = ({ className, asChild = false, ...props }) => {
    const Comp = asChild ? Slot : 'div'

    return (
        <Comp
            data-slot="sidebar-group-label"
            data-sidebar="group-label"
            className={cn(
                'flex h-8 shrink-0 items-center px-2 text-2xs font-medium uppercase tracking-caps text-ink-subtle',
                'transition-[margin,opacity] duration-200 ease-linear',
                'group-data-[collapsible=icon]:-mt-8 group-data-[collapsible=icon]:opacity-0',
                className,
            )}
            {...props}
        />
    )
}
