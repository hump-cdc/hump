import type { FC } from 'react'

import { Card } from '@/shared/ui/card'
import { NumberFlow } from '@/shared/ui/number-flow'
import { ProgressBar } from '@/shared/ui/progress-bar'
import { useSwitchViewValue } from '@/shared/utils'

import { aggregate } from '../../model/selectors'
import type { SnapshotState } from '../../model/types'

export const ProcessingOverall: FC = () => {
    const data = useSwitchViewValue<SnapshotState>()
    if (data?.state !== 'processing') return null

    const { tables } = data
    const { percent } = aggregate(tables)
    const doneCount = tables.filter((table) => table.status === 'done').length

    return (
        <Card separated={false} className="gap-4 p-4.5">
            <div className="flex flex-col gap-1.5">
                <span className="font-semibold text-ink text-xl">Snapshot in progress</span>
                <span className="text-base text-ink-subtle">
                    Filling the configured sinks with the current data from the tables.
                </span>
            </div>

            <div className="flex items-end justify-between gap-4">
                <div className="flex items-baseline gap-2">
                    <NumberFlow
                        value={percent}
                        kind="percent"
                        className="font-semibold text-4xl text-ink tracking-stat"
                    />
                    <span className="text-base text-ink-subtle">complete</span>
                </div>
                <span className="pb-1 font-medium text-base text-ink-subtle">
                    <NumberFlow value={doneCount} /> of <NumberFlow value={tables.length} />
                </span>
            </div>

            <ProgressBar value={percent} size="md" />
        </Card>
    )
}
