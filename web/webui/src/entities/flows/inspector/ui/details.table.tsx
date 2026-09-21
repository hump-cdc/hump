import type { FC } from 'react'

import { formatNumberCompact } from '@/shared/utils'

import type { SelectedItem } from '../model/types'
import { IdentityCard } from './identity-card'
import { InspectorMetrics } from './inspector-metrics'

/** Inspector details for a selected table node: name identity and table metrics. */
export const TableDetails: FC<{ selected: Extract<SelectedItem, { kind: 'table' }> }> = ({ selected }) => (
    <>
        <IdentityCard eyebrow="TABLE">
            <span className="font-medium font-mono text-ink text-md">{selected.data.name}</span>
        </IdentityCard>
        <InspectorMetrics
            throughput={`${formatNumberCompact(selected.data.ops.insert.count)}/s`}
            successRate={98.2}
            totalEvents="1.24M"
            failures="2"
        />
    </>
)
