import type { ComponentProps, FC } from 'react'

import { Combobox as ComboboxPrimitive } from '@base-ui/react/combobox'

import { cn } from '@/shared/utils'

type ComboboxChipsInputProps = ComponentProps<typeof ComboboxPrimitive.Input>

const ComboboxChipsInput: FC<ComboboxChipsInputProps> = ({ className, ...props }) => (
    <ComboboxPrimitive.Input
        data-slot="combobox-chips-input"
        className={cn(
            'h-full min-w-4 flex-1 bg-transparent text-base text-ink outline-none placeholder:text-ink-tertiary',
            className,
        )}
        {...props}
    />
)

export { ComboboxChipsInput }
