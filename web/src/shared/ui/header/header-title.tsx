import type { ComponentProps, FC } from 'react'

import { cn } from '@/shared/utils'

type HeaderTitleProps = ComponentProps<'h1'>

export const HeaderTitle: FC<HeaderTitleProps> = ({ className, ...props }) => (
    <h1
        data-slot="header-title"
        className={cn('truncate text-lg font-semibold tracking-heading text-ink', className)}
        {...props}
    />
)
