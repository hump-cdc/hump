import type { FC, HTMLAttributes, Ref } from 'react'

import { cn } from '@/shared/utils'

type CardContentProps = HTMLAttributes<HTMLDivElement> & {
    ref?: Ref<HTMLDivElement>
}

export const CardContent: FC<CardContentProps> = ({ className, ...props }) => (
    <div className={cn('flex-1', className)} {...props} />
)
