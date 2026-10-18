import type { FC } from 'react'

import { Loader2, Search, Table2, TriangleAlert } from 'lucide-react'

import { errorMessage } from '@/shared/api'
import { Badge } from '@/shared/ui/badge'
import { Checkbox } from '@/shared/ui/checkbox'
import { Mark } from '@/shared/ui/mark'
import { NumberFlow } from '@/shared/ui/number-flow'
import { Placeholder } from '@/shared/ui/placeholder'
import { Table, TableRow } from '@/shared/ui/table'
import { TextInput } from '@/shared/ui/text-input'

import { useAvailableTables, useSelectedTables, useTables, useTablesEstimate } from '../../model/tables'
import { TABLE_COLUMNS } from './tables.columns'

export const Tables: FC = () => {
    const { data: tables = [], isPending, isError, error } = useAvailableTables()
    const { selectedIds, wheres, filter, toggle, toggleAll, setWhere, setFilter } = useTables()
    const selectedCount = useSelectedTables().length
    const estimates = useTablesEstimate()

    const query = filter.trim().toLowerCase()
    const visibleTables = query ? tables.filter((table) => table.name.toLowerCase().includes(query)) : tables

    const visibleSelectedCount = visibleTables.filter((table) => selectedIds.has(table.id)).length
    const allChecked = visibleTables.length > 0 && visibleSelectedCount === visibleTables.length
    const selectAllState = allChecked ? true : visibleSelectedCount > 0 ? 'indeterminate' : false

    return (
        <Table
            variant="bordered"
            columns={TABLE_COLUMNS}
            selectable
            header={
                <div className="flex items-center gap-2.5">
                    Tables
                    <Badge variant="active" border>
                        <NumberFlow value={selectedCount} /> of <NumberFlow value={tables.length} />
                    </Badge>
                </div>
            }
            headerAside={
                <label
                    htmlFor="snapshot-select-all"
                    className="flex cursor-pointer select-none items-center gap-2 text-base text-ink-muted"
                >
                    <Checkbox
                        id="snapshot-select-all"
                        checked={selectAllState}
                        onCheckedChange={() => toggleAll(visibleTables.map((table) => table.id))}
                        disabled={visibleTables.length === 0}
                    />
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
            {isPending ? (
                <Placeholder className="p-12" icon={<Loader2 className="animate-spin" />} title="Loading tables…" />
            ) : isError ? (
                <Placeholder
                    className="p-12"
                    icon={<TriangleAlert />}
                    title="Failed to load tables"
                    description={errorMessage(error)}
                />
            ) : tables.length === 0 ? (
                <Placeholder className="p-12" icon={<Table2 />} title="No configured tables found" />
            ) : visibleTables.length === 0 ? (
                <p className="p-12 text-center text-base text-ink-subtle">
                    No tables match <Mark>{filter}</Mark>.
                </p>
            ) : (
                visibleTables.map((table) => (
                    <TableRow
                        key={table.id}
                        row={{
                            table,
                            selected: selectedIds.has(table.id),
                            estimate: estimates[table.id],
                            where: wheres[table.id] ?? table.suggestedWhere,
                            onWhereCommit: (where) => setWhere(table.id, where),
                        }}
                        columns={TABLE_COLUMNS}
                        selectable={{
                            id: `snapshot-table-${table.id}`,
                            checked: selectedIds.has(table.id),
                            onToggle: () => toggle(table.id),
                        }}
                    />
                ))
            )}
        </Table>
    )
}
