import type { FC } from 'react'

import { MetricCard } from '@/shared/ui/card'
import { formatDuration, formatNumberCompact } from '@/shared/utils'

import { aggregate, selectedTables } from '../model/selectors'
import { useSnapshot } from '../model/use-snapshot'

export const RunningStats: FC = () => {
    const { selectedIds, progress, elapsedSec } = useSnapshot()

    const tables = selectedTables(selectedIds)
    const { rowsRead, itemsSent } = aggregate(tables, progress)

    return (
        <div className="flex flex-col md:flex-row gap-2.5">
            <MetricCard title="Elapsed" value={formatDuration(elapsedSec)} className="flex-1" />
            <MetricCard title="Rows read" value={formatNumberCompact(rowsRead)} className="flex-1" />
            <MetricCard title="Items sent" value={formatNumberCompact(itemsSent)} className="flex-1" />
        </div>
    )
}
