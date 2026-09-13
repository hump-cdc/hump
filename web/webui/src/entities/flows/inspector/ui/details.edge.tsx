import type { FC } from 'react'

import { ArrowRight } from 'lucide-react'

import type { SelectedItem } from '../model/types'
import { IdentityCard } from './identity-card'
import { InspectorMetrics } from './inspector-metrics'

/** Inspector details for a selected edge: source→target identity and edge metrics. */
export const EdgeDetails: FC<{ selected: Extract<SelectedItem, { kind: 'edge' }> }> = ({ selected }) => (
    <>
        <IdentityCard eyebrow="EDGE">
            <div className="flex items-center gap-2">
                <span className="font-mono text-md font-medium text-ink">{selected.sourceLabel}</span>
                <ArrowRight className="size-3.5 shrink-0 text-primary" />
                <span className="font-mono text-md font-medium text-ink">{selected.targetLabel}</span>
            </div>
        </IdentityCard>
        <InspectorMetrics throughput="842/s" successRate={98.2} totalEvents="1.24M" failures="2" />
    </>
)
