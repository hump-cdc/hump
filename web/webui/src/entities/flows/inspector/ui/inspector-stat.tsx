import type { FC } from 'react'

import { cn } from '@/shared/utils'

/** Single headline figure tile used in the inspector metrics grid. */
export const InspectorStat: FC<{ label: string; value: string; tone?: 'default' | 'success' | 'danger' }> = ({
    label,
    value,
    tone = 'default',
}) => (
    <div className="flex flex-1 flex-col gap-1 rounded-lg border border-hairline bg-surface-2 p-3">
        <span className="font-medium text-2xs text-ink-tertiary">{label}</span>
        <span
            className={cn(
                'font-mono font-semibold text-base',
                tone === 'success' && 'text-success',
                tone === 'danger' && 'text-danger',
                tone === 'default' && 'text-ink',
            )}
        >
            {value}
        </span>
    </div>
)
