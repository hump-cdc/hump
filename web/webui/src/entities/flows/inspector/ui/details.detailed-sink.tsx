import type { FC } from 'react'

import type { SelectedItem } from '../model/types'
import { IdentityCard } from './identity-card'
import { InspectorMetrics } from './inspector-metrics'

/**
 * Inspector details for a sink selected in the detailed view: name, type and host, with
 * metrics rolled up from its targets (the worst success rate; failing if any target fails).
 */
export const DetailedSinkDetails: FC<{ selected: Extract<SelectedItem, { kind: 'detailed-sink' }> }> = ({
    selected,
}) => {
    const { targets } = selected.data
    const successRate = Math.min(...targets.map((target) => target.successRate))
    const hasError = targets.some((target) => target.health === 'error')

    return (
        <>
            <IdentityCard eyebrow="SINK">
                <div className="flex flex-col gap-0.5">
                    <span className="font-medium font-mono text-ink text-md">{selected.data.name}</span>
                    <span className="text-ink-subtle text-xs">
                        {[selected.data.type, selected.data.host].filter(Boolean).join(' · ')}
                    </span>
                </div>
            </IdentityCard>
            <InspectorMetrics
                throughput="842/s"
                successRate={targets.length > 0 ? successRate : 100}
                totalEvents={selected.data.count}
                failures={hasError ? '37' : '2'}
            />
        </>
    )
}
