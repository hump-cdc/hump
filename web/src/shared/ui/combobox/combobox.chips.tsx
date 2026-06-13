import type { ComponentProps, FC } from 'react'

import { Combobox as ComboboxPrimitive } from '@base-ui/react/combobox'
import { X } from 'lucide-react'

import { cn } from '@/shared/utils'

type ComboboxChipsProps = ComponentProps<typeof ComboboxPrimitive.Chips>

const ComboboxChips: FC<ComboboxChipsProps> = ({ className, ...props }) => (
    <ComboboxPrimitive.Chips
        data-slot="combobox-chips"
        className={cn(
            'flex h-9 w-full flex-nowrap items-center gap-1.5 overflow-hidden rounded-lg border border-hairline bg-canvas px-2.5 text-base outline-none transition-[color,box-shadow]',
            'focus-within:outline-none focus-within:ring-2 focus-within:ring-primary focus-within:ring-offset-2 focus-within:ring-offset-canvas',
            'aria-invalid:border-danger aria-invalid:focus-within:ring-danger',
            'has-data-disabled:pointer-events-none has-data-disabled:opacity-50',
            className,
        )}
        {...props}
    />
)

type ComboboxChipProps = ComponentProps<typeof ComboboxPrimitive.Chip> & { showRemove?: boolean }

const ComboboxChip: FC<ComboboxChipProps> = ({ className, children, showRemove = true, ...props }) => (
    <ComboboxPrimitive.Chip
        data-slot="combobox-chip"
        className={cn(
            'flex h-5.5 shrink-0 items-center gap-1 rounded-md bg-primary-soft pl-2 text-sm font-medium whitespace-nowrap text-primary select-none',
            showRemove ? 'pr-1' : 'pr-2',
            className,
        )}
        {...props}
    >
        {children}
        {showRemove && (
            <ComboboxPrimitive.ChipRemove
                data-slot="combobox-chip-remove"
                aria-label="Remove"
                className="flex size-4 items-center justify-center rounded text-primary/70 outline-none transition-colors hover:text-primary focus-visible:ring-2 focus-visible:ring-primary/40"
            >
                <X className="size-3" />
            </ComboboxPrimitive.ChipRemove>
        )}
    </ComboboxPrimitive.Chip>
)

export { ComboboxChip, ComboboxChips }
