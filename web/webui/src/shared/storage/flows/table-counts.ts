import { db } from '../db'
import type { FlowTableDto } from './types'
import { writeFlows } from './utils/write-flows'

/** Per-table CDC counters read from the scrape. Absolute values, not increments. */
export type FlowTableCounts = Pick<
    FlowTableDto,
    'schema' | 'table' | 'insertCount' | 'updateCount' | 'deleteCount' | 'truncateCount'
>

/**
 * Replace the per-table CDC counters with the values the scrape reports. The backend counts each
 * row once whatever the sink fan-out, so these are authoritative.
 */
export const saveTableCounts = (rows: FlowTableCounts[]): Promise<void> =>
    writeFlows(async (resolve) => {
        for (const row of rows) {
            const id = await resolve.table(row.schema, row.table)
            const existing = await db.flowTables.get(id)
            if (existing) await db.flowTables.put({ ...existing, ...row })
        }
    })
