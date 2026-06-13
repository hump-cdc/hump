import type { FC } from 'react'

import type { SelectedItem } from '../model/types'
import { IdentityCard } from './identity-card'
import { InspectorMetrics } from './inspector-metrics'

/** Inspector details for a selected sink node: name + optional subtitle and sink metrics. */
export const SinkDetails: FC<{ selected: Extract<SelectedItem, { kind: 'sink' }> }> = ({ selected }) => (
    <>
        <IdentityCard eyebrow="SINK">
            <div className="flex flex-col gap-0.5">
                <span className="font-mono text-md font-medium text-ink">{selected.data.name}</span>
                {selected.data.subtitle ? (
                    <span className="text-xs text-ink-subtle">{selected.data.subtitle}</span>
                ) : null}
            </div>
        </IdentityCard>
        <InspectorMetrics
            throughput={selected.data.rate}
            successRate={selected.data.successRate}
            totalEvents="1.24M"
            failures={selected.data.health === 'error' ? '37' : '2'}
        />
    </>
)
