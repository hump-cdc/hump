import type { FC } from 'react'

import { cn, formatTime } from '@/shared/utils'

import type { SelectedItem } from '../model/types'
import { useRecentOperations } from '../model/use-recent-operations'

/** Newest deliveries of the selected node or edge (desktop-only in the inspector). */
export const RecentOperations: FC<{ selected: SelectedItem }> = ({ selected }) => {
    const { operations, isLoading } = useRecentOperations(selected)

    return (
        <div className="flex flex-col">
            <span className="pb-1 font-semibold text-2xs text-ink-tertiary tracking-widest">RECENT OPERATIONS</span>
            {operations.map((operation) => (
                <div key={operation.id} className="flex items-center gap-2.5 border-hairline border-b py-2">
                    <span className={cn('size-1.5 rounded-full', operation.isSuccess ? 'bg-success' : 'bg-danger')} />
                    <span className="font-medium font-mono text-ink-muted text-xs">{operation.op}</span>
                    <span className="flex-1" />
                    <span className="font-mono text-2xs text-ink-tertiary">{formatTime(operation.time)}</span>
                </div>
            ))}
            {!isLoading && operations.length === 0 && (
                <span className="py-2 text-ink-tertiary text-xs">No operations yet</span>
            )}
        </div>
    )
}
