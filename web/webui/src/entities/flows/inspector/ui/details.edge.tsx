import type { FC } from 'react'

import { InfoCard } from '@/shared/ui/card'
import { ProgressChart } from '@/shared/ui/charts/progress-chart'
import { NumberFlow } from '@/shared/ui/number-flow'

import { resolveDelivery } from '../model/resolve'
import type { SelectedItem } from '../model/types'
import { useInspectorTopology } from '../model/use-inspector-topology'

/**
 * Inspector details for a selected edge: its source table, its target with the owning sink, and the
 * deliveries along it. A summary edge has no single target, so it names the sink instead.
 */
export const EdgeDetails: FC<{ selected: Extract<SelectedItem, { kind: 'edge' }> }> = ({ selected }) => {
    const topology = useInspectorTopology()
    const stats = topology && resolveDelivery(topology, selected)

    return (
        <>
            <InfoCard title="Source">{selected.sourceLabel}</InfoCard>
            <InfoCard title="Target" footer={[selected.type, selected.host].filter(Boolean).join(' · ')}>
                {selected.target ?? selected.targetLabel}
            </InfoCard>
            <div className="grid grid-cols-2 gap-2">
                <InfoCard title="Total events" className="col-span-2">
                    <NumberFlow value={stats?.operations} kind="compact" />
                </InfoCard>
                <InfoCard title="Success" variant="success">
                    <NumberFlow value={stats?.success} kind="compact" />
                </InfoCard>
                <InfoCard title="Failures" variant="failed">
                    <NumberFlow value={stats?.failure} kind="compact" />
                </InfoCard>
            </div>

            <ProgressChart title="Success rate" value={stats?.successRate} showError />
        </>
    )
}
