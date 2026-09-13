import type { FC } from 'react'

import { MetricCard } from '@/shared/ui/card'
import { formatDuration, formatNumber } from '@/shared/utils'

import { aggregate, selectedTables } from '../model/selectors'
import { useSnapshot } from '../model/use-snapshot'

export const CompleteStats: FC = () => {
    const { selectedIds, progress, elapsedSec } = useSnapshot()

    const tables = selectedTables(selectedIds)
    const { rowsRead, itemsSent } = aggregate(tables, progress)

    return (
        <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
            <MetricCard title="Tables" value={tables.length} />
            <MetricCard title="Rows read" value={formatNumber(rowsRead)} />
            <MetricCard title="Items sent" value={formatNumber(itemsSent)} />
            <MetricCard title="Duration" value={formatDuration(elapsedSec)} />
        </div>
    )
}
