import { db } from '../db'
import { bumpEdge } from './utils/bump-edge'
import { writeFlows } from './utils/write-flows'

/** `count` operations that flowed from `schema.table` to `target` on the `sinkType`/`host` sink. */
export type FlowOperationDelta = {
    schema: string
    table: string
    sinkType: string
    host: string
    target: string
    count: number
}

/**
 * Bump the sink, the target and both edges by the counts in `deltas`, creating any node the
 * configuration did not declare. `p95`, `successRate` and the `flow_tables` counters belong to
 * the metrics scrape and are never touched here.
 */
export const saveOperationDeltas = (deltas: FlowOperationDelta[]): Promise<void> =>
    writeFlows(async (resolve) => {
        for (const delta of deltas) {
            const tableId = await resolve.table(delta.schema, delta.table)
            const sinkId = await resolve.sink(delta.sinkType, delta.host)
            const targetId = await resolve.target(sinkId, delta.target)

            const sink = await db.flowSinks.get(sinkId)
            if (sink) await db.flowSinks.put({ ...sink, operationCount: sink.operationCount + delta.count })

            const target = await db.flowSinkTargets.get(targetId)
            if (target) await db.flowSinkTargets.put({ ...target, operationCount: target.operationCount + delta.count })

            await bumpEdge(
                db.flowConnectionTableSink,
                '[tableId+sinkId]',
                [tableId, sinkId],
                { tableId, sinkId, operationCount: 0 },
                delta.count,
            )
            await bumpEdge(
                db.flowConnectionTableTarget,
                '[tableId+targetId]',
                [tableId, targetId],
                { tableId, targetId, operationCount: 0 },
                delta.count,
            )
        }
    })
