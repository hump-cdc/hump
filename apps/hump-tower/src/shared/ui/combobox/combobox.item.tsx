import type { ComponentProps, FC } from 'react'

import { Combobox as ComboboxPrimitive } from '@base-ui/react/combobox'

import { cn } from '@/shared/utils'

type ComboboxItemProps = ComponentProps<typeof ComboboxPrimitive.Item>

const ComboboxItem: FC<ComboboxItemProps> = ({ className, ...props }) => (
    <ComboboxPrimitive.Item
        data-slot="combobox-item"
        className={cn(
            'group/combobox-item flex w-full cursor-default select-none items-center gap-2.5 rounded-md px-2 py-2 text-md outline-none',
            'data-disabled:pointer-events-none data-highlighted:bg-surface-1 data-disabled:opacity-50',
            className,
        )}
        {...props}
    />
)

export { ComboboxItem }
