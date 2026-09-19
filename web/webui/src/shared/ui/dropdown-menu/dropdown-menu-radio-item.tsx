import type { ComponentProps, FC } from 'react'

import {
    ItemIndicator as DropdownMenuItemIndicator,
    RadioItem as DropdownMenuRadioItemPrimitive,
} from '@radix-ui/react-dropdown-menu'
import { Circle } from 'lucide-react'

import { cn } from '@/shared/utils'

type DropdownMenuRadioItemProps = ComponentProps<typeof DropdownMenuRadioItemPrimitive>

export const DropdownMenuRadioItem: FC<DropdownMenuRadioItemProps> = ({ className, children, ...props }) => (
    <DropdownMenuRadioItemPrimitive
        data-slot="dropdown-menu-radio-item"
        className={cn(
            'relative flex cursor-default select-none items-center gap-2 rounded-md py-1.5 pr-2 pl-8 text-ink-muted text-sm outline-none transition-colors data-[disabled]:pointer-events-none data-[highlighted]:bg-surface-1 data-[highlighted]:text-ink data-[disabled]:opacity-50 [&_svg]:shrink-0',
            className,
        )}
        {...props}
    >
        <span className="absolute left-2 flex size-4 items-center justify-center">
            <DropdownMenuItemIndicator>
                <Circle className="size-2 fill-current" />
            </DropdownMenuItemIndicator>
        </span>
        {children}
    </DropdownMenuRadioItemPrimitive>
)
