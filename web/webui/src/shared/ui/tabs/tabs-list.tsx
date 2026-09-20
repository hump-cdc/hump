import { type ComponentProps, type FC, useRef } from 'react'

import { List as TabsPrimitiveList } from '@radix-ui/react-tabs'
import { cva, type VariantProps } from 'class-variance-authority'

import { cn } from '@/shared/utils'

import { TabsListIndicator } from './tabs-list-indicator'

const tabsListVariants = cva('group/tabs-list relative items-center', {
    variants: {
        variant: {
            pill: 'inline-flex w-fit rounded-lg border border-hairline bg-surface-1 p-0.5',
            underline: 'flex w-full border-hairline border-b',
        },
    },
    defaultVariants: {
        variant: 'pill',
    },
})

type TabsListProps = ComponentProps<typeof TabsPrimitiveList> & VariantProps<typeof tabsListVariants>

export const TabsList: FC<TabsListProps> = ({ className, variant = 'pill', children, ...props }) => {
    const listRef = useRef<HTMLDivElement>(null)

    return (
        <TabsPrimitiveList
            ref={listRef}
            data-slot="tabs-list"
            data-variant={variant}
            className={cn(tabsListVariants({ variant, className }))}
            {...props}
        >
            <TabsListIndicator listRef={listRef} variant={variant ?? 'pill'} />
            {children}
        </TabsPrimitiveList>
    )
}
