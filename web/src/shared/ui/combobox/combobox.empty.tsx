import type { ComponentProps, FC } from 'react'

import { Combobox as ComboboxPrimitive } from '@base-ui/react/combobox'

import { cn } from '@/shared/utils'

type ComboboxEmptyProps = ComponentProps<typeof ComboboxPrimitive.Empty>

const ComboboxEmpty: FC<ComboboxEmptyProps> = ({ className, ...props }) => (
    <ComboboxPrimitive.Empty
        data-slot="combobox-empty"
        className={cn(
            'hidden w-full justify-center px-2 py-6 text-center text-base text-ink-subtle group-data-empty/combobox-content:flex',
            className,
        )}
        {...props}
    />
)

export { ComboboxEmpty }
