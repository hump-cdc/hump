import type { ComponentProps, FC } from 'react'

import type { Operation } from '@/entities/operations/list/@x/details'

import { PropertyRow } from './operation-details.property-row'

/** Where the operation came from and went to: table, sink, host and target. */
export const Details: FC<{ operation: Operation; size?: ComponentProps<typeof PropertyRow>['size'] }> = ({
    operation,
    size,
}) => (
    <div className="flex flex-col">
        <PropertyRow size={size} label="Table" value={operation.table} copyValue={operation.table} />
        <PropertyRow size={size} label="Sink" value={operation.sink} copyValue={operation.sink} />
        <PropertyRow
            size={size}
            label="Host"
            value={operation.sinkHost}
            copyValue={operation.sinkHost}
            valueClassName="text-ink-muted"
        />
        <PropertyRow
            size={size}
            label="Target"
            value={operation.target}
            copyValue={operation.target}
            valueClassName="text-ink-muted"
        />
    </div>
)
