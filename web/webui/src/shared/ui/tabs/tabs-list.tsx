import { type ComponentProps, type FC, useRef } from 'react'

import { List as TabsPrimitiveList } from '@radix-ui/react-tabs'

import { cn } from '@/shared/utils'

import { TabsListIndicator } from './tabs-list-indicator'

type TabsListProps = ComponentProps<typeof TabsPrimitiveList>

export const TabsList: FC<TabsListProps> = ({ className, children, ...props }) => {
    const listRef = useRef<HTMLDivElement>(null)

    return (
        <TabsPrimitiveList
            ref={listRef}
            data-slot="tabs-list"
            className={cn(
                'relative inline-flex w-fit items-center rounded-lg border border-hairline bg-surface-1 p-0.5',
                className,
            )}
            {...props}
        >
            <TabsListIndicator listRef={listRef} />
            {children}
        </TabsPrimitiveList>
    )
}
