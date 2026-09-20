import type { FC } from 'react'

import type { SelectedItem } from '../model/types'
import { IdentityCard } from './identity-card'
import { InspectorMetrics } from './inspector-metrics'

/** Inspector details for a selected sink node: name + sink type and sink metrics. */
export const SinkDetails: FC<{ selected: Extract<SelectedItem, { kind: 'sink' }> }> = ({ selected }) => (
    <>
        <IdentityCard eyebrow="SINK">
            <div className="flex flex-col gap-0.5">
                <span className="font-medium font-mono text-ink text-md">{selected.data.name}</span>
                <span className="text-ink-subtle text-xs">{selected.data.type}</span>
            </div>
        </IdentityCard>
        <InspectorMetrics
            throughput="842/s"
            successRate={selected.data.successRate}
            totalEvents={selected.data.operations}
            failures={selected.data.health === 'error' ? '37' : '2'}
        />
    </>
)
