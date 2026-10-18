import type { FC } from 'react'

import { Card, CardHeader } from '@/shared/ui/card'
import { ProgressBar } from '@/shared/ui/progress-bar'
import { Sink } from '@/shared/ui/sink'
import { formatNumber, useSwitchViewValue } from '@/shared/utils'

import { aggregate } from '../../model/selectors'
import type { SinkBreakdown, SnapshotState, Table } from '../../model/types'

export const DoneSinks: FC = () => {
    const data = useSwitchViewValue<SnapshotState>()
    if (data?.state !== 'done') return null

    const { itemsSent } = aggregate(data.tables)
    const sinks = sinkBreakdown(data.tables)
    const maxSinkItems = sinks.reduce((max, sink) => Math.max(max, sink.items), 0)
    const showProgress = sinks.length > 1

    return (
        <Card className="xl:w-90">
            <CardHeader
                aside={<span className="font-medium text-ink-subtle text-sm">{formatNumber(itemsSent)} total</span>}
            >
                <span className="font-semibold text-ink text-md">Items per sink</span>
            </CardHeader>

            <div className="flex flex-col gap-4 p-4">
                {sinks.map((sink) => (
                    <div key={`${sink.type} ${sink.host}`} className="flex flex-col gap-2">
                        <div className="flex items-center justify-between gap-2">
                            <Sink type={sink.type} host={sink.host} className="min-w-0" />
                            <span className="font-mono text-ink-muted text-sm">{formatNumber(sink.items)}</span>
                        </div>
                        {showProgress && (
                            <ProgressBar value={maxSinkItems === 0 ? 0 : (sink.items / maxSinkItems) * 100} size="sm" />
                        )}
                    </div>
                ))}
            </div>
        </Card>
    )
}

/** Items delivered per sink connection, summed over every table delivered to it. */
const sinkBreakdown = (tables: Table[]): SinkBreakdown[] => {
    const totals = new Map<string, SinkBreakdown>()

    for (const table of tables) {
        for (const sink of table.sinks) {
            const key = `${sink.type} ${sink.host}`
            const entry = totals.get(key) ?? { type: sink.type, host: sink.host, items: 0 }
            entry.items += sink.count
            totals.set(key, entry)
        }
    }

    return [...totals.values()].sort((a, b) => b.items - a.items)
}
