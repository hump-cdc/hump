import type { FC } from 'react'

import { Badge } from '@/shared/ui/badge'
import { formatLatency, formatTime } from '@/shared/utils'

import type { Operation } from '../model/types'

type OperationRetriesProps = {
    row: Operation
}

export const OperationRetries: FC<OperationRetriesProps> = ({ row }) => (
    <table className="w-full border-collapse">
        <thead>
            <tr className="border-hairline border-b bg-surface-1 text-left font-semibold text-2xs text-ink-subtle uppercase tracking-caps">
                {['Attempt', 'Started', 'Duration', 'Result', 'Error'].map((h) => (
                    <th key={h} className="h-8 px-3.5 font-semibold">
                        {h}
                    </th>
                ))}
            </tr>
        </thead>
        <tbody>
            {row.retries?.map((a, index) => (
                <tr
                    key={a.attempt}
                    className={index < (row.retries?.length ?? 0) - 1 ? 'border-hairline border-b' : undefined}
                >
                    <td className="h-11 px-3.5 font-mono text-ink-muted">#{a.attempt}</td>
                    <td className="h-11 px-3.5 font-mono text-ink-subtle text-sm">{formatTime(a.started)}</td>
                    <td className="h-11 px-3.5 font-mono text-ink-muted">{formatLatency(a.durationMs)}</td>
                    <td className="h-11 px-3.5">
                        <Badge variant={a.result === 'failed' ? 'failed' : 'healthy'}>{a.result}</Badge>
                    </td>
                    <td className="h-11 px-3.5 text-danger text-sm">{a.error}</td>
                </tr>
            ))}
        </tbody>
    </table>
)
