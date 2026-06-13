import type { ComponentProps, FC } from 'react'

import { Trigger as SheetTriggerPrimitive } from '@radix-ui/react-dialog'

type SheetTriggerProps = ComponentProps<typeof SheetTriggerPrimitive>

export const SheetTrigger: FC<SheetTriggerProps> = (props) => (
    <SheetTriggerPrimitive data-slot="sheet-trigger" {...props} />
)
