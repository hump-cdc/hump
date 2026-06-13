import type { FC } from 'react'

import { Badge } from '@/shared/ui/badge'
import { formatTime } from '@/shared/utils'

import type { Operation } from '../model/types'

export const OperationRetries: FC<{ row: Operation }> = ({ row }) => (
    <table className="w-full border-collapse">
        <thead>
            <tr className="border-b border-hairline bg-surface-1 text-left text-2xs font-semibold tracking-caps text-ink-subtle uppercase">
                {['Attempt', 'Started', 'Duration', 'Result', 'Error'].map((h) => (
                    <th key={h} className="px-3.5 py-2 font-semibold">
                        {h}
                    </th>
                ))}
            </tr>
        </thead>
        <tbody>
            {row.retries?.map((a, index) => (
                <tr
                    key={a.attempt}
                    className={index < (row.retries?.length ?? 0) - 1 ? 'border-b border-hairline' : undefined}
                >
                    <td className="px-3.5 py-2.5 font-mono text-ink-muted">#{a.attempt}</td>
                    <td className="px-3.5 py-2.5 font-mono text-sm text-ink-subtle">{formatTime(a.started)}</td>
                    <td className="px-3.5 py-2.5 font-mono text-ink-muted">{a.duration}</td>
                    <td className="px-3.5 py-2.5">
                        <Badge variant={a.result === 'failed' ? 'failed' : 'healthy'}>{a.result}</Badge>
                    </td>
                    <td className="px-3.5 py-2.5 text-sm text-danger">{a.error}</td>
                </tr>
            ))}
        </tbody>
    </table>
)
