import type {
    SnapshotDoneEventModel,
    SnapshotDoneTableModel,
    SnapshotIdleEventModel,
    SnapshotProcessingEventModel,
    SnapshotProcessingTableModel,
} from '@/shared/api'

/** A sink a table is delivered to, with the target rendered for that table. */
export type SnapshotSink = {
    type: string
    host: string
    target: string
}

/** A table column, with its type as PostgreSQL formats it. */
export type SnapshotColumn = {
    name: string
    type: string
}

/** A configured table that can be snapshotted. */
export type SnapshotTable = {
    id: string
    schema: string
    table: string
    name: string
    suggestedWhere: string
    columns: SnapshotColumn[]
    sinks: SnapshotSink[]
}

/** Row count of a selected table under its current filter. */
export type TableEstimate =
    | { status: 'loading' }
    | {
          status: 'ready'
          count: number
          rough: boolean
          fetching: boolean
      }
    | {
          status: 'error'
          error: string
      }

/**
 * Phase of one table of a run: `wait`, `processing` and `done` while the run is in progress,
 * `success` or `error` once it is finished.
 */
export type TableStatus = SnapshotProcessingTableModel['status'] | SnapshotDoneTableModel['status']

/** A sink one table of a run is delivered to, with the rows it received so far. */
export type Sink = {
    type: string
    host: string
    target: string
    count: number
}

/** One table of a run, as the server reports it. */
export type Table = {
    id: string
    schema: string
    table: string
    name: string
    /** Filter the rows are read with; empty reads every row. */
    where: string
    status: TableStatus
    error?: string
    rowsRead: number
    totalRows: number
    sinks: Sink[]
}

/** State of the server's snapshot runner, one run at a time. */
export type SnapshotState =
    | Pick<SnapshotIdleEventModel, 'state'>
    | (Pick<SnapshotProcessingEventModel, 'state'> & { startedAt: Date; tables: Table[] })
    | (Pick<SnapshotDoneEventModel, 'state'> & { startedAt: Date; finishedAt: Date; tables: Table[] })

/** Aggregated totals across all tables of a run. */
export type SnapshotAggregate = {
    totalRows: number
    rowsRead: number
    itemsSent: number
    percent: number
}

/** Items delivered to one sink, summed over all snapshotted tables. */
export type SinkBreakdown = {
    type: string
    host: string
    items: number
}
