import { db } from '../db'
import type { FlowSinkStats } from './sink-stats'
import { mergeStats } from './utils/merge-stats'
import { writeFlows } from './utils/write-flows'

/** Delivery stats for one target within a sink, identified by `[sinkId+target]`. */
export type FlowTargetStats = FlowSinkStats & { target: string }

/**
 * Replace the per-target delivery stats with the values derived from the scrape, creating targets —
 * and their parent sink — the operations stream has not seen. The scrape covers everything since
 * the process started, so this fills in targets active before the page was opened.
 */
export const saveTargetStats = (rows: FlowTargetStats[]): Promise<void> =>
    writeFlows(async (resolve) => {
        for (const row of rows) {
            const sinkId = await resolve.sink(row.sinkType, row.host)
            const id = await resolve.target(sinkId, row.target)
            const existing = await db.flowSinkTargets.get(id)
            if (existing) await db.flowSinkTargets.put(mergeStats(existing, row))
        }
    })
