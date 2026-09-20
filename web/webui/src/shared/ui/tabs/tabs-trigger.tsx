import type { ComponentProps, FC } from 'react'

import { Trigger as TabsPrimitiveTrigger } from '@radix-ui/react-tabs'

import { cn } from '@/shared/utils'

type TabsTriggerProps = ComponentProps<typeof TabsPrimitiveTrigger>

export const TabsTrigger: FC<TabsTriggerProps> = ({ className, ...props }) => (
    <TabsPrimitiveTrigger
        data-slot="tabs-trigger"
        className={cn(
            'relative z-10 inline-flex cursor-pointer items-center justify-center gap-1.5 whitespace-nowrap',
            'rounded-md px-3 py-1.5 font-medium text-ink-subtle text-sm outline-none transition-colors',
            'hover:text-ink-muted disabled:pointer-events-none disabled:opacity-50',
            'data-[state=active]:text-ink',
            // `underline` list: triggers share the width equally, the active one set in semibold.
            'group-data-[variant=underline]/tabs-list:flex-1 group-data-[variant=underline]/tabs-list:rounded-none',
            'group-data-[variant=underline]/tabs-list:px-2 group-data-[variant=underline]/tabs-list:pt-3 group-data-[variant=underline]/tabs-list:pb-2.5',
            'group-data-[variant=underline]/tabs-list:text-base group-data-[variant=underline]/tabs-list:data-[state=active]:font-semibold',
            'focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-canvas',
            '[&_svg:not([class*=size-])]:size-4 [&_svg]:pointer-events-none [&_svg]:shrink-0',
            className,
        )}
        {...props}
    />
)
