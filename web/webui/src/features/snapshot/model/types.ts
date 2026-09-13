/** Lifecycle of the snapshot feature: pick tables → run → review results. */
export type SnapshotState = 'initial' | 'processing' | 'finished'

/** Per-table progress phase while a snapshot is running. */
export type TableStatus = 'waiting' | 'processing' | 'done'

/** A table that can be snapshotted, with its estimated size and target sinks. */
export type SnapshotTable = {
    id: string
    name: string
    /** Estimated total rows to re-read. */
    totalRows: number
    /** Sinks every row of this table is re-streamed to. */
    sinks: string[]
}

/** Runtime progress for a single table during/after a run. */
export type TableProgress = {
    id: string
    status: TableStatus
    rowsRead: number
    /** Rows fanned out across sinks (rowsRead × sink count). */
    itemsSent: number
}

/** Aggregated totals across all selected tables. */
export type SnapshotAggregate = {
    totalRows: number
    rowsRead: number
    itemsSent: number
    /** Completion in the 0–100 range. */
    percent: number
}

/** Items delivered to one sink, summed over all snapshotted tables. */
export type SinkBreakdown = {
    name: string
    items: number
}
