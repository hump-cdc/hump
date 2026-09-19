import type { ComponentProps, FC } from 'react'

import { Slot } from '@radix-ui/react-slot'
import { cva, type VariantProps } from 'class-variance-authority'

import { Tooltip, TooltipContent, TooltipTrigger } from '@/shared/ui/tooltip'
import { cn } from '@/shared/utils'

import { useSidebar } from './sidebar-context'

const sidebarMenuButtonVariants = cva(
    'peer/menu-button flex w-full cursor-pointer items-center gap-2.5 overflow-hidden rounded-lg p-2 text-left text-ink-muted text-md leading-5 outline-none ring-primary transition-colors' +
        'hover:bg-surface-2 hover:text-ink focus-visible:ring-2 active:bg-surface-2 disabled:pointer-events-none disabled:opacity-50 aria-disabled:pointer-events-none aria-disabled:opacity-50' +
        'data-[active=true]:bg-primary-soft data-[active=true]:font-medium data-[active=true]:text-primary data-[active=true]:hover:bg-primary-soft data-[active=true]:hover:text-primary' +
        'group-data-[collapsible=icon]:size-9! group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:p-2! group-data-[collapsible=icon]:[&>span:last-child]:hidden' +
        '[&>span:last-child]:truncate [&>svg]:size-4.5 [&>svg]:shrink-0 [&>svg]:text-ink-subtle data-[active=true]:[&>svg]:text-primary',
    {
        variants: {
            size: {
                default: 'h-9',
                sm: 'h-8 text-base',
                lg: 'h-11',
            },
        },
        defaultVariants: {
            size: 'default',
        },
    },
)

type SidebarMenuButtonProps = ComponentProps<'button'> &
    VariantProps<typeof sidebarMenuButtonVariants> & {
        asChild?: boolean
        isActive?: boolean
        tooltip?: string | ComponentProps<typeof TooltipContent>
    }

export const SidebarMenuButton: FC<SidebarMenuButtonProps> = ({
    asChild = false,
    isActive = false,
    size = 'default',
    tooltip,
    className,
    ...props
}) => {
    const Comp = asChild ? Slot : 'button'
    const { isMobile, state } = useSidebar()

    const button = (
        <Comp
            data-slot="sidebar-menu-button"
            data-sidebar="menu-button"
            data-size={size}
            data-active={isActive}
            className={cn(sidebarMenuButtonVariants({ size }), className)}
            {...props}
        />
    )

    if (!tooltip) return button

    const tooltipProps = typeof tooltip === 'string' ? { children: tooltip } : tooltip

    return (
        <Tooltip>
            <TooltipTrigger asChild>{button}</TooltipTrigger>
            <TooltipContent side="right" align="center" hidden={state !== 'collapsed' || isMobile} {...tooltipProps} />
        </Tooltip>
    )
}
