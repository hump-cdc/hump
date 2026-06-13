import type { ComponentProps, FC } from 'react'

import { Combobox as ComboboxPrimitive } from '@base-ui/react/combobox'
import { X } from 'lucide-react'

import { cn } from '@/shared/utils'

type ComboboxClearProps = ComponentProps<typeof ComboboxPrimitive.Clear>

const ComboboxClear: FC<ComboboxClearProps> = ({ className, ...props }) => (
    <ComboboxPrimitive.Clear
        data-slot="combobox-clear"
        aria-label="Clear selection"
        className={cn(
            'flex size-5 shrink-0 items-center justify-center rounded text-ink-subtle outline-none transition-colors hover:text-ink-muted focus-visible:ring-2 focus-visible:ring-primary/40',
            className,
        )}
        {...props}
    >
        <X className="size-3.5" />
    </ComboboxPrimitive.Clear>
)

export { ComboboxClear }
