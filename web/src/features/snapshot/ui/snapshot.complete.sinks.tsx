import type { FC } from 'react'

import { Card, CardHeader } from '@/shared/ui/card'
import { ProgressBar } from '@/shared/ui/progress-bar'
import { formatNumber } from '@/shared/utils'

import { aggregate, selectedTables, sinkBreakdown } from '../model/selectors'
import { useSnapshot } from '../model/use-snapshot'

export const CompleteSinks: FC = () => {
    const { selectedIds, progress } = useSnapshot()

    const tables = selectedTables(selectedIds)
    const { itemsSent } = aggregate(tables, progress)
    const sinks = sinkBreakdown(tables, progress)
    const maxSinkItems = sinks.reduce((max, sink) => Math.max(max, sink.items), 0)

    return (
        <Card className="lg:w-90">
            <CardHeader
                aside={<span className="text-sm font-medium text-ink-subtle">{formatNumber(itemsSent)} total</span>}
            >
                <span className="text-md font-semibold text-ink">Items per sink</span>
            </CardHeader>

            <div className="flex flex-col gap-4 p-4">
                {sinks.map((sink) => (
                    <div key={sink.name} className="flex flex-col gap-2">
                        <div className="flex items-center justify-between gap-2">
                            <span className="font-mono text-base text-ink">{sink.name}</span>
                            <span className="font-mono text-sm text-ink-muted">{formatNumber(sink.items)}</span>
                        </div>
                        <ProgressBar value={maxSinkItems === 0 ? 0 : (sink.items / maxSinkItems) * 100} size="sm" />
                    </div>
                ))}
            </div>
        </Card>
    )
}
