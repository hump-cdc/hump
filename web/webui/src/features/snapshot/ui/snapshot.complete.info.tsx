import { type FC, useState } from 'react'

import { CircleCheck } from 'lucide-react'

import { formatDateTime } from '@/shared/utils'

import { selectedTables } from '../model/selectors'
import { useSnapshot } from '../model/use-snapshot'

export const CompleteInfo: FC = () => {
    const { selectedIds } = useSnapshot()
    const [finishedAt] = useState(() => new Date())

    const tables = selectedTables(selectedIds)

    return (
        <div className="flex items-center gap-3.5 rounded-xl border border-success/30 bg-success-soft p-5">
            <CircleCheck className="size-6 shrink-0 text-success" />
            <div className="flex flex-col gap-0.5">
                <span className="font-semibold text-ink text-md">Snapshot complete</span>
                <span className="text-base text-ink-subtle">
                    All {tables.length} {tables.length === 1 ? 'table' : 'tables'} re-read and streamed to sinks ·
                    finished {formatDateTime(finishedAt)}
                </span>
            </div>
        </div>
    )
}
