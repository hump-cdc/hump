import type { ComponentProps, FC } from 'react'

import type { MetricsTable } from '@/entities/metrics/@x/inspector'
import { Badge } from '@/shared/ui/badge'
import { InfoCard } from '@/shared/ui/card'
import { NumberFlow } from '@/shared/ui/number-flow'

import { resolveTable, sumTableOps } from '../model/resolve'
import type { SelectedItem } from '../model/types'
import { useInspectorTopology } from '../model/use-inspector-topology'

/** Inspector details for a selected table: name, rows replicated in total and per operation with their p95. */
export const TableDetails: FC<{ selected: Extract<SelectedItem, { kind: 'table' }> }> = ({ selected }) => {
    const topology = useInspectorTopology()
    const table = topology && resolveTable(topology, selected.name)

    return (
        <>
            <InfoCard title="Table">{selected.name}</InfoCard>

            <div className="grid grid-cols-1 gap-2">
                <InfoCard title="Total events">
                    <NumberFlow value={table && sumTableOps(table)} kind="compact" />
                </InfoCard>
            </div>

            <div className="grid grid-cols-2 gap-2">
                <OperationCard title="Insert" variant="success" op={table?.insert} />
                <OperationCard title="Update" variant="neutral" op={table?.update} />
                <OperationCard title="Delete" variant="failed" op={table?.delete} />
                <OperationCard title="Truncate" variant="neutral" op={table?.truncate} />
            </div>
        </>
    )
}

const OperationCard: FC<{
    title: string
    variant: ComponentProps<typeof InfoCard>['variant']
    op?: MetricsTable['insert']
}> = ({ title, variant, op }) => (
    <InfoCard title={title} variant={variant}>
        <div className="flex items-center justify-between gap-2">
            <NumberFlow value={op?.count} kind="compact" />
            {op?.p95 !== undefined && (
                <Badge className="min-h-5 font-mono text-2xs">
                    <NumberFlow value={op.p95} kind="latency" />
                </Badge>
            )}
        </div>
    </InfoCard>
)
