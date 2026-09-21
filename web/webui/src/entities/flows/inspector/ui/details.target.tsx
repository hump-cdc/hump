import type { FC } from 'react'

import { formatNumberCompact } from '@/shared/utils'

import type { SelectedItem } from '../model/types'
import { IdentityCard } from './identity-card'
import { InspectorMetrics } from './inspector-metrics'

/** Inspector details for a selected sink target: name, kind and owning sink, with target metrics. */
export const TargetDetails: FC<{ selected: Extract<SelectedItem, { kind: 'target' }> }> = ({ selected }) => (
    <>
        <IdentityCard eyebrow="TARGET">
            <div className="flex flex-col gap-0.5">
                <span className="font-medium font-mono text-ink text-md">{selected.data.name}</span>
                <span className="text-ink-subtle text-xs">
                    {selected.data.kind} · {selected.sinkLabel}
                </span>
            </div>
        </IdentityCard>
        <InspectorMetrics
            throughput="842/s"
            successRate={selected.data.successRate}
            totalEvents={formatNumberCompact(selected.data.count)}
            failures={selected.data.health === 'error' ? '37' : '2'}
        />
    </>
)
