import type { FC } from 'react'

import { Badge } from '@/shared/ui/badge'
import { Table, TableRow } from '@/shared/ui/table'
import { useSwitchViewValue } from '@/shared/utils'

import type { SnapshotState } from '../../model/types'
import { PROGRESS_COLUMNS } from './tables.columns'

export const ProcessingTables: FC = () => {
    const data = useSwitchViewValue<SnapshotState>()
    if (data?.state !== 'processing') return null

    const { tables } = data

    return (
        <Table
            variant="bordered"
            columns={PROGRESS_COLUMNS}
            header={
                <div className="flex items-center gap-2.5">
                    <span className="font-semibold text-ink text-md">Tables</span>
                    <Badge variant="neutral">{tables.length}</Badge>
                </div>
            }
            headerAside={
                <div className="flex items-center gap-3.5 text-ink-subtle text-sm">
                    <span className="flex items-center gap-1.5">
                        <span className="size-1.5 rounded-full bg-ink-subtle" />
                        wait
                    </span>
                    <span className="flex items-center gap-1.5">
                        <span className="size-1.5 rounded-full bg-primary" />
                        processing
                    </span>
                    <span className="flex items-center gap-1.5">
                        <span className="size-1.5 rounded-full bg-success" />
                        done
                    </span>
                </div>
            }
        >
            {tables.map((table) => (
                <TableRow key={table.id} row={{ table }} columns={PROGRESS_COLUMNS} />
            ))}
        </Table>
    )
}
