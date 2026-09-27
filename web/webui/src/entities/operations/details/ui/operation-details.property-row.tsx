import type { FC, ReactNode } from 'react'

import { CopyButton } from '@/shared/ui/copy-button'
import { cn } from '@/shared/utils'

type PropertyRowProps = {
    label: string
    /** Rendered at the row's right edge; a string renders in mono. */
    value: ReactNode
    /** Clipboard text; omit to hide the copy button. */
    copyValue?: string
    /** `compact` suits the side drawer, `comfortable` the mobile sheet. */
    size?: 'compact' | 'comfortable'
    /** Width class of the label column. */
    labelClassName?: string
    valueClassName?: string
}

/** One label/value line, separated from the next by a hairline. */
export const PropertyRow: FC<PropertyRowProps> = ({
    label,
    value,
    copyValue,
    size = 'compact',
    labelClassName,
    valueClassName,
}) => (
    <div
        className={cn(
            'flex items-center gap-2.5 border-hairline border-b',
            size === 'compact' ? 'py-1.75 text-sm' : 'py-2.5 text-base',
        )}
    >
        <span className={cn('shrink-0 truncate text-ink-subtle', labelClassName ?? 'w-17.5')}>{label}</span>
        {typeof value === 'string' ? (
            <span
                title={value}
                className={cn('min-w-0 flex-1 truncate text-right font-medium font-mono text-ink', valueClassName)}
            >
                {value}
            </span>
        ) : (
            <div className={cn('flex min-w-0 flex-1 justify-end', valueClassName)}>{value}</div>
        )}
        {copyValue !== undefined && <CopyButton value={copyValue} label={`Copy ${label}`} />}
    </div>
)
