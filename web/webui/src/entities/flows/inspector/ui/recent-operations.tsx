import type { FC } from 'react'

import { cn } from '@/shared/utils'

/** Recent-operation rows shown in the activity list — representative demo data. */
const RECENT_OPERATIONS = [
    { id: 'op-0', op: 'INSERT', time: '12:04:51', ok: true },
    { id: 'op-1', op: 'INSERT', time: '12:04:51', ok: true },
    { id: 'op-2', op: 'INSERT', time: '12:04:50', ok: true },
    { id: 'op-3', op: 'INSERT', time: '12:04:49', ok: false },
    { id: 'op-4', op: 'INSERT', time: '12:04:48', ok: true },
]

/** Compact recent-operations activity list (desktop-only in the inspector). */
export const RecentOperations: FC = () => (
    <div className="flex flex-col">
        <span className="pb-1 font-semibold text-2xs text-ink-tertiary tracking-widest">RECENT OPERATIONS</span>
        {RECENT_OPERATIONS.map((entry) => (
            <div key={entry.id} className="flex items-center gap-2.5 border-hairline border-b py-2">
                <span className={cn('size-1.5 rounded-full', entry.ok ? 'bg-success' : 'bg-danger')} />
                <span className="font-medium font-mono text-ink-muted text-xs">{entry.op}</span>
                <span className="flex-1" />
                <span className="font-mono text-2xs text-ink-tertiary">{entry.time}</span>
            </div>
        ))}
    </div>
)
