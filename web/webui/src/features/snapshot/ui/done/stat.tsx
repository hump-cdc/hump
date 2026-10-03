import type { FC } from 'react'

import { MetricCard } from '@/shared/ui/card'
import { NumberFlow } from '@/shared/ui/number-flow'
import { useSwitchViewValue } from '@/shared/utils'

import { aggregate } from '../../model/selectors'
import type { SnapshotState } from '../../model/types'

export const DoneStats: FC = () => {
    const data = useSwitchViewValue<SnapshotState>()
    if (data?.state !== 'done') return null

    const { rowsRead, itemsSent } = aggregate(data.tables)
    const durationSec = (data.finishedAt.getTime() - data.startedAt.getTime()) / 1000

    return (
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
            <MetricCard title="Elapsed" value={<NumberFlow value={durationSec} kind="duration" />} />
            <MetricCard title="Rows read" value={<NumberFlow value={rowsRead} kind="integer" />} />
            <MetricCard title="Events sent" value={<NumberFlow value={itemsSent} kind="integer" />} />
        </div>
    )
}
