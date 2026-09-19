import type { ComponentProps, FC } from 'react'

import {
    CheckboxItem as DropdownMenuCheckboxItemPrimitive,
    ItemIndicator as DropdownMenuItemIndicator,
} from '@radix-ui/react-dropdown-menu'
import { Check } from 'lucide-react'

import { cn } from '@/shared/utils'

type DropdownMenuCheckboxItemProps = ComponentProps<typeof DropdownMenuCheckboxItemPrimitive>

export const DropdownMenuCheckboxItem: FC<DropdownMenuCheckboxItemProps> = ({
    className,
    children,
    checked,
    ...props
}) => (
    <DropdownMenuCheckboxItemPrimitive
        data-slot="dropdown-menu-checkbox-item"
        checked={checked}
        className={cn(
            'relative flex cursor-default select-none items-center gap-2 rounded-md py-1.5 pr-2 pl-8 text-ink-muted text-sm outline-none transition-colors data-[disabled]:pointer-events-none data-[highlighted]:bg-surface-1 data-[highlighted]:text-ink data-[disabled]:opacity-50 [&_svg]:size-4 [&_svg]:shrink-0',
            className,
        )}
        {...props}
    >
        <span className="absolute left-2 flex size-4 items-center justify-center">
            <DropdownMenuItemIndicator>
                <Check className="size-4" strokeWidth={3} />
            </DropdownMenuItemIndicator>
        </span>
        {children}
    </DropdownMenuCheckboxItemPrimitive>
)
