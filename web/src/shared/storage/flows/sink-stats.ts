import { db } from '../db'
import type { FlowDeliveryStats, FlowSinkDto } from './types'
import { mergeStats } from './utils/merge-stats'
import { writeFlows } from './utils/write-flows'

/** Delivery stats for one sink endpoint, identified by `[sinkType+host]`. */
export type FlowSinkStats = FlowDeliveryStats & { sinkType: string; host: string }

/** Read every persisted sink node — a single-store read, so live queries on it ignore other flow writes. */
export const getFlowSinks = (): Promise<FlowSinkDto[]> => db.flowSinks.toArray()

/** Replace the per-sink delivery stats with the values derived from the scrape. */
export const saveSinkStats = (rows: FlowSinkStats[]): Promise<void> =>
    writeFlows(async (resolve) => {
        for (const row of rows) {
            const id = await resolve.sink(row.sinkType, row.host)
            const existing = await db.flowSinks.get(id)
            if (existing) await db.flowSinks.put(mergeStats(existing, row))
        }
    })
