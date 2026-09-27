import type { FC, ReactElement } from 'react'

import { Tooltip, TooltipContent, TooltipTrigger } from '@/shared/ui/tooltip'

type MetricHintProps = {
    /** Explanation shown on hover. */
    hint: string
    /** The metric element; receives the trigger props, so it must forward them to a DOM node. */
    children: ReactElement
}

/** Wraps a node metric so hovering it explains what the number means. */
export const MetricHint: FC<MetricHintProps> = ({ hint, children }) => (
    <Tooltip>
        <TooltipTrigger asChild>{children}</TooltipTrigger>
        <TooltipContent className="max-w-56 text-pretty">{hint}</TooltipContent>
    </Tooltip>
)
