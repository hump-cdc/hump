import type { ComponentProps, FC } from 'react'

import { Trigger as TabsPrimitiveTrigger } from '@radix-ui/react-tabs'

import { cn } from '@/shared/utils'

type TabsTriggerProps = ComponentProps<typeof TabsPrimitiveTrigger>

export const TabsTrigger: FC<TabsTriggerProps> = ({ className, ...props }) => (
    <TabsPrimitiveTrigger
        data-slot="tabs-trigger"
        className={cn(
            'relative z-10 inline-flex cursor-pointer items-center justify-center gap-1.5 whitespace-nowrap',
            'rounded-md px-3 py-1.5 text-sm font-medium text-ink-subtle outline-none transition-colors',
            'hover:text-ink-muted disabled:pointer-events-none disabled:opacity-50',
            'data-[state=active]:text-ink',
            'focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-canvas',
            '[&_svg:not([class*=size-])]:size-4 [&_svg]:pointer-events-none [&_svg]:shrink-0',
            className,
        )}
        {...props}
    />
)
