import type { FC, HTMLAttributes, Ref } from 'react'

import { cn } from '@/shared/utils'

type MarkProps = HTMLAttributes<HTMLElement> & {
    ref?: Ref<HTMLElement>
}

export const Mark: FC<MarkProps> = ({ className, ...props }) => (
    <mark
        data-slot="mark"
        className={cn('rounded-md border border-hairline bg-surface-3 px-1 py-0.5 font-mono text-ink', className)}
        {...props}
    />
)
