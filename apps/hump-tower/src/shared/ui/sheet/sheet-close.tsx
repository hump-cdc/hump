import type { ComponentProps, FC } from 'react'

import { Close as SheetClosePrimitive } from '@radix-ui/react-dialog'

type SheetCloseProps = ComponentProps<typeof SheetClosePrimitive>

export const SheetClose: FC<SheetCloseProps> = (props) => <SheetClosePrimitive data-slot="sheet-close" {...props} />
