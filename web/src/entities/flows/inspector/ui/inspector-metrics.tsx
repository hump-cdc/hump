import type { FC } from 'react'

import { InspectorStat } from './inspector-stat'

/** Headline figures grid plus the success-rate bar; values are supplied per selection kind. */
export const InspectorMetrics: FC<{
    throughput: string
    successRate: number
    totalEvents: string
    failures: string
}> = ({ throughput, successRate, totalEvents, failures }) => {
    const successLabel = `${successRate}%`

    return (
        <>
            <div className="flex flex-col gap-2">
                <div className="flex gap-2">
                    <InspectorStat label="Throughput" value={throughput} />
                    <InspectorStat label="Success" value={successLabel} tone="success" />
                </div>
                <div className="flex gap-2">
                    <InspectorStat label="Total events" value={totalEvents} />
                    <InspectorStat label="Failures" value={failures} tone="danger" />
                </div>
            </div>

            <div className="flex flex-col gap-2">
                <div className="flex items-center justify-between">
                    <span className="text-xs font-medium text-ink-subtle">Success rate</span>
                    <span className="font-mono text-xs font-medium text-success">{successLabel}</span>
                </div>
                <div className="flex h-2 w-full gap-0.5 overflow-hidden rounded-full bg-surface-3">
                    <div
                        className="h-full bg-success"
                        style={{ width: `${Math.max(0, Math.min(100, successRate))}%` }}
                    />
                    <div className="h-full flex-1 bg-danger" />
                </div>
            </div>
        </>
    )
}
