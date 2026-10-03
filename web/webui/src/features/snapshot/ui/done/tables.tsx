import type { FC } from 'react'

import { Table, TableRow } from '@/shared/ui/table'
import { useSwitchViewValue } from '@/shared/utils'

import type { SnapshotState } from '../../model/types'
import { RESULT_COLUMNS } from './tables.columns'

export const DoneTables: FC = () => {
    const data = useSwitchViewValue<SnapshotState>()
    if (data?.state !== 'done') return null

    const { tables } = data
    const failedCount = tables.filter((table) => table.status === 'error').length

    return (
        <Table
            variant="bordered"
            columns={RESULT_COLUMNS}
            className="lg:flex-1"
            header={<span className="font-semibold text-ink text-md">Snapshotted tables</span>}
            headerAside={
                <span className="font-medium text-ink-subtle text-sm">
                    {tables.length} {tables.length === 1 ? 'table' : 'tables'}
                    {failedCount > 0 && ` · ${failedCount} incomplete`}
                </span>
            }
        >
            {tables.map((table) => (
                <TableRow
                    key={table.id}
                    row={{ table }}
                    columns={RESULT_COLUMNS}
                    mobileZoneClasses={{ meta: 'pl-6', badges: 'pl-6' }}
                />
            ))}
        </Table>
    )
}
