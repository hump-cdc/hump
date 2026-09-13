import type { ComponentProps, FC } from 'react'

import { Item as DropdownMenuItemPrimitive } from '@radix-ui/react-dropdown-menu'
import { cva, type VariantProps } from 'class-variance-authority'

import { cn } from '@/shared/utils'

const dropdownMenuItemVariants = cva(
    'relative flex cursor-default select-none items-center gap-2 rounded-md px-2 py-1.5 text-sm outline-none transition-colors data-[disabled]:pointer-events-none data-[disabled]:opacity-50 [&_svg]:size-4 [&_svg]:shrink-0',
    {
        variants: {
            variant: {
                default: 'text-ink-muted data-[highlighted]:bg-surface-1 data-[highlighted]:text-ink',
                destructive: 'text-danger data-[highlighted]:bg-danger-soft data-[highlighted]:text-danger',
            },
            inset: {
                false: '',
                true: 'pl-8',
            },
        },
        defaultVariants: {
            variant: 'default',
            inset: false,
        },
    },
)

type DropdownMenuItemProps = ComponentProps<typeof DropdownMenuItemPrimitive> &
    VariantProps<typeof dropdownMenuItemVariants>

export const DropdownMenuItem: FC<DropdownMenuItemProps> = ({ className, variant, inset, ...props }) => (
    <DropdownMenuItemPrimitive
        data-slot="dropdown-menu-item"
        className={cn(dropdownMenuItemVariants({ variant, inset, className }))}
        {...props}
    />
)
