import type { ComponentProps, FC } from 'react'

import { Root as SheetRoot } from '@radix-ui/react-dialog'

type SheetProps = ComponentProps<typeof SheetRoot>

export const Sheet: FC<SheetProps> = (props) => <SheetRoot data-slot="sheet" {...props} />
