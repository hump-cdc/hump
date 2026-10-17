import type { FC } from 'react'

import { CircleCheck, CircleX } from 'lucide-react'

import { cn, formatDateTimeRange, useSwitchViewValue } from '@/shared/utils'

import type { SnapshotState } from '../../model/types'

export const DoneInfo: FC = () => {
    const data = useSwitchViewValue<SnapshotState>()
    if (data?.state !== 'done') return null

    const { tables, startedAt, finishedAt } = data
    const hasFailed = tables.some((table) => table.status === 'error')

    return (
        <div
            className={cn(
                'flex items-center gap-3.5 rounded-xl border p-5',
                hasFailed ? 'border-danger/30 bg-danger-soft' : 'border-success/30 bg-success-soft',
            )}
        >
            {hasFailed ? (
                <CircleX className="size-6 shrink-0 text-danger" />
            ) : (
                <CircleCheck className="size-6 shrink-0 text-success" />
            )}
            <div className="flex flex-col gap-0.5">
                <span className="font-semibold text-ink text-md">
                    {hasFailed ? 'Some tables were not read completely' : 'Completed successfully'}
                </span>
                <span className="text-base text-ink-subtle">{formatDateTimeRange(startedAt, finishedAt)}</span>
            </div>
        </div>
    )
}
