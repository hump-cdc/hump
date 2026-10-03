import type { SnapshotAggregate, SnapshotTable, Table } from './types'

/** Query the snapshot reads a table with, as shown to the user. */
export const snapshotQuery = (table: Pick<SnapshotTable, 'name'>, where: string): string => {
    const lines = ['SELECT *', `FROM   ${table.name}`]
    if (where.trim()) lines.push(`WHERE  ${where.trim()}`)

    return lines.join('\n')
}

/** Roll up per-table progress into overall totals and completion percent. */
export const aggregate = (tables: Table[]): SnapshotAggregate => {
    const totalRows = tables.reduce((total, table) => total + table.totalRows, 0)
    const rowsRead = tables.reduce((sum, table) => sum + table.rowsRead, 0)
    const items = tables.reduce((sum, table) => sum + table.sinks.reduce((acc, sink) => acc + sink.count, 0), 0)
    const percent = totalRows === 0 ? 0 : Math.min(100, (rowsRead / totalRows) * 100)

    return { totalRows, rowsRead, itemsSent: items, percent }
}
