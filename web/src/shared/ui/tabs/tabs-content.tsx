import type { ComponentProps, FC } from 'react'

import { Content as TabsPrimitiveContent } from '@radix-ui/react-tabs'

import { cn } from '@/shared/utils'

type TabsContentProps = ComponentProps<typeof TabsPrimitiveContent>

export const TabsContent: FC<TabsContentProps> = ({ className, ...props }) => (
    <TabsPrimitiveContent data-slot="tabs-content" className={cn('flex-1 outline-none', className)} {...props} />
)
