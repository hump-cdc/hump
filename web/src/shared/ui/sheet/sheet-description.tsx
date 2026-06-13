import type { ComponentProps, FC } from 'react'

import { Description as SheetDescriptionPrimitive } from '@radix-ui/react-dialog'

import { cn } from '@/shared/utils'

type SheetDescriptionProps = ComponentProps<typeof SheetDescriptionPrimitive>

export const SheetDescription: FC<SheetDescriptionProps> = ({ className, ...props }) => (
    <SheetDescriptionPrimitive
        data-slot="sheet-description"
        className={cn('text-sm text-ink-muted', className)}
        {...props}
    />
)
