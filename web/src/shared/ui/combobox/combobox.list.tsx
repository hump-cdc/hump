import type { ComponentProps, FC } from 'react'

import { Combobox as ComboboxPrimitive } from '@base-ui/react/combobox'

import { cn } from '@/shared/utils'

type ComboboxListProps = ComponentProps<typeof ComboboxPrimitive.List>

const ComboboxList: FC<ComboboxListProps> = ({ className, ...props }) => (
    <ComboboxPrimitive.List
        data-slot="combobox-list"
        className={cn('max-h-80 scroll-py-1 overflow-y-auto p-1.5', className)}
        {...props}
    />
)

export { ComboboxList }
