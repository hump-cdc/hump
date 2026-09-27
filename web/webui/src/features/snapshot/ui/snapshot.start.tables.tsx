import type { FC } from 'react'

import { Search } from 'lucide-react'

import { Badge } from '@/shared/ui/badge'
import { Checkbox } from '@/shared/ui/checkbox'
import type { TableRowColumn } from '@/shared/ui/table'
import { Table, TableRow } from '@/shared/ui/table'
import { TextInput } from '@/shared/ui/text-input'
import { formatNumber } from '@/shared/utils'

import { SNAPSHOT_TABLES } from '../model/mock'
import type { SnapshotTable } from '../model/types'
import { useSnapshot } from '../model/use-snapshot'

const SELECT_COLUMNS: TableRowColumn<SnapshotTable>[] = [
    {
        key: 'name',
        header: 'Table',
        className: 'flex-1',
        cell: (t) => <span className="min-w-0 flex-1 truncate font-mono text-base text-ink">{t.name}</span>,
    },
    {
        key: 'rows',
        header: 'Est. rows',
        className: 'w-32.5',
        cell: (t) => (
            <span className="font-mono text-base text-ink-muted lg:w-32.5">
                <span className="text-ink-subtle lg:hidden">≈ </span>
                {formatNumber(t.totalRows)}
            </span>
        ),
    },
    {
        key: 'sinks',
        header: 'Sinks',
        className: 'w-57.5',
        cell: (t) => (
            <div className="flex flex-wrap gap-1.5 lg:w-57.5">
                {t.sinks.map((sink) => (
                    <Badge key={sink} variant="neutral" className="font-mono">
                        {sink}
                    </Badge>
                ))}
            </div>
        ),
    },
]

export const StartTables: FC = () => {
    const { selectedIds, filter, toggle, toggleAll, setFilter } = useSnapshot()

    const query = filter.trim().toLowerCase()
    const visibleTables = query
        ? SNAPSHOT_TABLES.filter((table) => table.name.toLowerCase().includes(query))
        : SNAPSHOT_TABLES

    const selectedCount = selectedIds.size
    const allChecked = selectedCount === SNAPSHOT_TABLES.length
    const selectAllState = allChecked ? true : selectedCount > 0 ? 'indeterminate' : false

    return (
        <Table
            variant="bordered"
            columns={SELECT_COLUMNS}
            selectable
            header={
                <div className="flex items-center gap-2.5">
                    <span className="font-semibold text-ink text-lg tracking-heading">Select tables</span>
                    <Badge variant="active" border>
                        {selectedCount} of {SNAPSHOT_TABLES.length}
                    </Badge>
                </div>
            }
            headerAside={
                <label
                    htmlFor="snapshot-select-all"
                    className="flex cursor-pointer select-none items-center gap-2 text-base text-ink-muted"
                >
                    <Checkbox id="snapshot-select-all" checked={selectAllState} onCheckedChange={toggleAll} />
                    <span>Select all</span>
                </label>
            }
            toolbar={
                <TextInput
                    icon={<Search />}
                    font="sans"
                    placeholder="Filter tables…"
                    value={filter}
                    onValueChange={setFilter}
                />
            }
        >
            {visibleTables.length > 0 ? (
                visibleTables.map((table) => (
                    <TableRow
                        key={table.id}
                        row={table}
                        columns={SELECT_COLUMNS}
                        selectable={{
                            id: `snapshot-table-${table.id}`,
                            checked: selectedIds.has(table.id),
                            onToggle: () => toggle(table.id),
                        }}
                    />
                ))
            ) : (
                <p className="px-4 py-8 text-center text-base text-ink-subtle">No tables match “{filter}”.</p>
            )}
        </Table>
    )
}
