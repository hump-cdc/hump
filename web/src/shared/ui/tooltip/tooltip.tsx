import type { ComponentProps, FC } from 'react'

import { Root as TooltipRoot } from '@radix-ui/react-tooltip'

type TooltipProps = ComponentProps<typeof TooltipRoot>

export const Tooltip: FC<TooltipProps> = (props) => <TooltipRoot data-slot="tooltip" {...props} />
