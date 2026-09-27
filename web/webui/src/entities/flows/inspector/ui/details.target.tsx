import type { FC } from 'react'

import { InfoCard } from '@/shared/ui/card'
import { ProgressChart } from '@/shared/ui/charts/progress-chart'
import { NumberFlow } from '@/shared/ui/number-flow'

import { resolveDelivery } from '../model/resolve'
import type { SelectedItem } from '../model/types'
import { useInspectorTopology } from '../model/use-inspector-topology'

/** Inspector details for a selected sink target: its owning sink and name, with its delivery figures. */
export const TargetDetails: FC<{ selected: Extract<SelectedItem, { kind: 'target' }> }> = ({ selected }) => {
    const topology = useInspectorTopology()
    const stats = topology && resolveDelivery(topology, selected)

    return (
        <>
            <InfoCard title="Sink">{selected.sinkLabel}</InfoCard>
            <InfoCard title="Target">{selected.name}</InfoCard>

            <div className="grid grid-cols-2 gap-2">
                <InfoCard title="Total events">
                    <NumberFlow value={stats?.operations} kind="compact" />
                </InfoCard>
                <InfoCard title="Latency (p95)">
                    <NumberFlow value={stats?.p95} kind="latency" />
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
