import type { ComponentProps, FC } from 'react'

import { Content as TooltipContentPrimitive, Portal as TooltipPortal } from '@radix-ui/react-tooltip'
import { cva, type VariantProps } from 'class-variance-authority'

import { cn } from '@/shared/utils'

const tooltipContentVariants = cva(
    [
        'z-50 w-fit select-none rounded-md px-2.5 py-1 font-medium text-xs shadow-md',
        'fade-in-0 zoom-in-95 data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95 animate-in data-[state=closed]:animate-out',
        'data-[side=bottom]:slide-in-from-top-1 data-[side=left]:slide-in-from-right-1 data-[side=right]:slide-in-from-left-1 data-[side=top]:slide-in-from-bottom-1',
    ],
    {
        variants: {
            variant: {
                default: 'bg-ink text-canvas',
                primary: 'bg-primary text-on-primary',
            },
        },
        defaultVariants: {
            variant: 'default',
        },
    },
)

type TooltipContentProps = ComponentProps<typeof TooltipContentPrimitive> & VariantProps<typeof tooltipContentVariants>

export const TooltipContent: FC<TooltipContentProps> = ({ className, variant, sideOffset = 4, children, ...props }) => (
    <TooltipPortal>
        <TooltipContentPrimitive
            data-slot="tooltip-content"
            sideOffset={sideOffset}
            className={cn(tooltipContentVariants({ variant, className }))}
            {...props}
        >
            {children}
        </TooltipContentPrimitive>
    </TooltipPortal>
)
