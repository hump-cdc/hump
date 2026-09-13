import type { ComponentProps, FC } from 'react'

import { Root as TabsPrimitiveRoot } from '@radix-ui/react-tabs'

import { cn } from '@/shared/utils'

type TabsProps = ComponentProps<typeof TabsPrimitiveRoot>

export const Tabs: FC<TabsProps> = ({ className, ...props }) => (
    <TabsPrimitiveRoot data-slot="tabs" className={cn('flex flex-col gap-2', className)} {...props} />
)
