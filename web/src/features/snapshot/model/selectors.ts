import { SNAPSHOT_TABLES } from './mock'
import type { SinkBreakdown, SnapshotAggregate, SnapshotTable, TableProgress } from './types'

/** Selected tables in canonical display order. */
export const selectedTables = (selectedIds: Set<string>): SnapshotTable[] =>
    SNAPSHOT_TABLES.filter((table) => selectedIds.has(table.id))

/** Sum of estimated rows across the given tables. */
export const estimatedRows = (tables: SnapshotTable[]): number =>
    tables.reduce((total, table) => total + table.totalRows, 0)

/** Roll up per-table progress into overall totals and completion percent. */
export const aggregate = (tables: SnapshotTable[], progress: Record<string, TableProgress>): SnapshotAggregate => {
    const totalRows = estimatedRows(tables)
    const rowsRead = tables.reduce((sum, table) => sum + (progress[table.id]?.rowsRead ?? 0), 0)
    const itemsSent = tables.reduce((sum, table) => sum + (progress[table.id]?.itemsSent ?? 0), 0)
    const percent = totalRows === 0 ? 0 : (rowsRead / totalRows) * 100

    return { totalRows, rowsRead, itemsSent, percent }
}

/** Items delivered per sink, distributed evenly across each table's sinks. */
export const sinkBreakdown = (tables: SnapshotTable[], progress: Record<string, TableProgress>): SinkBreakdown[] => {
    const totals = new Map<string, number>()

    for (const table of tables) {
        const rowsRead = progress[table.id]?.rowsRead ?? 0
        for (const sink of table.sinks) {
            totals.set(sink, (totals.get(sink) ?? 0) + rowsRead)
        }
    }

    return [...totals.entries()].map(([name, items]) => ({ name, items })).sort((a, b) => b.items - a.items)
}
