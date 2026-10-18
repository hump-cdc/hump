import type { ComponentProps, FC } from 'react'

import { Provider as TooltipProviderPrimitive } from '@radix-ui/react-tooltip'

type TooltipProviderProps = ComponentProps<typeof TooltipProviderPrimitive>

export const TooltipProvider: FC<TooltipProviderProps> = ({ delayDuration = 0, ...props }) => (
    <TooltipProviderPrimitive data-slot="tooltip-provider" delayDuration={delayDuration} {...props} />
)
