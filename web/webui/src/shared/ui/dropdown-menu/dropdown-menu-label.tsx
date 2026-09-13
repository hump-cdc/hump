import type { ComponentProps, FC } from 'react'

import { Label as DropdownMenuLabelPrimitive } from '@radix-ui/react-dropdown-menu'
import { cva, type VariantProps } from 'class-variance-authority'

import { cn } from '@/shared/utils'

const dropdownMenuLabelVariants = cva('px-2 py-1.5 text-sm font-medium text-ink-subtle', {
    variants: {
        inset: {
            false: '',
            true: 'pl-8',
        },
    },
    defaultVariants: {
        inset: false,
    },
})

type DropdownMenuLabelProps = ComponentProps<typeof DropdownMenuLabelPrimitive> &
    VariantProps<typeof dropdownMenuLabelVariants>

export const DropdownMenuLabel: FC<DropdownMenuLabelProps> = ({ className, inset, ...props }) => (
    <DropdownMenuLabelPrimitive
        data-slot="dropdown-menu-label"
        className={cn(dropdownMenuLabelVariants({ inset, className }))}
        {...props}
    />
)
