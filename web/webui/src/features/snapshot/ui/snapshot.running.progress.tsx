import type { FC } from 'react'

import { Card } from '@/shared/ui/card'
import { ProgressBar } from '@/shared/ui/progress-bar'

import { aggregate, selectedTables } from '../model/selectors'
import { useSnapshot } from '../model/use-snapshot'

export const RunningProgress: FC = () => {
    const { selectedIds, progress } = useSnapshot()

    const tables = selectedTables(selectedIds)
    const { percent } = aggregate(tables, progress)
    const doneCount = tables.filter((table) => progress[table.id]?.status === 'done').length

    return (
        <Card separated={false} className="gap-4 p-4.5">
            <div className="flex flex-col gap-1.5">
                <span className="font-semibold text-ink text-xl">Snapshot in progress</span>
                <span className="text-base text-ink-subtle">
                    Re-reading all tables and re-streaming every row to the configured sinks.
                </span>
            </div>

            <div className="flex items-end justify-between gap-4">
                <div className="flex items-baseline gap-2">
                    <span className="font-mono font-semibold text-4xl text-ink tracking-stat">
                        {Math.round(percent)}%
                    </span>
                    <span className="text-base text-ink-subtle">complete</span>
                </div>
                <span className="pb-1 font-medium text-base text-ink-subtle">
                    {doneCount} of {tables.length} done
                </span>
            </div>

            <ProgressBar value={percent} size="md" />
        </Card>
    )
}
