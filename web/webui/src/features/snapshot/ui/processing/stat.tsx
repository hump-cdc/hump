import type { FC } from 'react'

import { MetricCard } from '@/shared/ui/card'
import { ElapsedTime } from '@/shared/ui/elapsed-time'
import { NumberFlow } from '@/shared/ui/number-flow'
import { useSwitchViewValue } from '@/shared/utils'

import { aggregate } from '../../model/selectors'
import type { SnapshotState } from '../../model/types'

export const ProcessingStats: FC = () => {
    const data = useSwitchViewValue<SnapshotState>()
    if (data?.state !== 'processing') return null

    const { rowsRead, itemsSent } = aggregate(data.tables)

    return (
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
            <MetricCard title="Elapsed" value={<ElapsedTime startedAt={data.startedAt} />} />
            <MetricCard title="Rows read" value={<NumberFlow value={rowsRead} kind="compact" />} />
            <MetricCard title="Events sent" value={<NumberFlow value={itemsSent} kind="compact" />} />
        </div>
    )
}
