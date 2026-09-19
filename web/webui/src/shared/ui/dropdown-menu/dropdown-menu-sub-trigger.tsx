import type { ComponentProps, FC } from 'react'

import { SubTrigger as DropdownMenuSubTriggerPrimitive } from '@radix-ui/react-dropdown-menu'
import { cva, type VariantProps } from 'class-variance-authority'
import { ChevronRight } from 'lucide-react'

import { cn } from '@/shared/utils'

const dropdownMenuSubTriggerVariants = cva(
    'flex cursor-default select-none items-center gap-2 rounded-md px-2 py-1.5 text-ink-muted text-sm outline-none transition-colors data-[disabled]:pointer-events-none data-[highlighted]:bg-surface-1 data-[state=open]:bg-surface-1 data-[highlighted]:text-ink data-[state=open]:text-ink data-[disabled]:opacity-50 [&_svg]:size-4 [&_svg]:shrink-0',
    {
        variants: {
            inset: {
                false: '',
                true: 'pl-8',
            },
        },
        defaultVariants: {
            inset: false,
        },
    },
)

type DropdownMenuSubTriggerProps = ComponentProps<typeof DropdownMenuSubTriggerPrimitive> &
    VariantProps<typeof dropdownMenuSubTriggerVariants>

export const DropdownMenuSubTrigger: FC<DropdownMenuSubTriggerProps> = ({ className, inset, children, ...props }) => (
    <DropdownMenuSubTriggerPrimitive
        data-slot="dropdown-menu-sub-trigger"
        className={cn(dropdownMenuSubTriggerVariants({ inset, className }))}
        {...props}
    >
        {children}
        <ChevronRight className="ml-auto" />
    </DropdownMenuSubTriggerPrimitive>
)
