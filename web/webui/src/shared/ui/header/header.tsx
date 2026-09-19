import type { ComponentProps, FC } from 'react'

import { cn } from '@/shared/utils'

type HeaderProps = ComponentProps<'header'>

export const Header: FC<HeaderProps> = ({ className, ...props }) => (
    <header
        data-slot="header"
        className={cn(
            'sticky top-0 z-10 flex h-14 w-full shrink-0 items-center justify-between gap-3 border-hairline border-b bg-canvas px-4 lg:px-6',
            className,
        )}
        {...props}
    />
)
