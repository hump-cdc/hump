import type { ComponentProps, FC } from 'react'

import { Title as SheetTitlePrimitive } from '@radix-ui/react-dialog'

import { cn } from '@/shared/utils'

type SheetTitleProps = ComponentProps<typeof SheetTitlePrimitive>

export const SheetTitle: FC<SheetTitleProps> = ({ className, ...props }) => (
    <SheetTitlePrimitive data-slot="sheet-title" className={cn('font-semibold text-ink', className)} {...props} />
)
