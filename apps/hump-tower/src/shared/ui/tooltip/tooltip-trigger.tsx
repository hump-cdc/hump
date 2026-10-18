import type { ComponentProps, FC } from 'react'

import { Trigger as TooltipTriggerPrimitive } from '@radix-ui/react-tooltip'

type TooltipTriggerProps = ComponentProps<typeof TooltipTriggerPrimitive>

export const TooltipTrigger: FC<TooltipTriggerProps> = (props) => (
    <TooltipTriggerPrimitive data-slot="tooltip-trigger" {...props} />
)
