import type { ComponentProps, FC } from 'react'

import { cn } from '@/shared/utils'

type SheetHeaderProps = ComponentProps<'div'>

export const SheetHeader: FC<SheetHeaderProps> = ({ className, ...props }) => (
    <div data-slot="sheet-header" className={cn('flex flex-col gap-1.5 p-4', className)} {...props} />
)
