import type { ConfigResponseBodyModel } from '@/shared/api'

import { db } from '../db'
import { FLOW_STORES } from './schema'
import type { FlowSinkDto, FlowSinkTargetDto, FlowTableDto, FlowTableSinkDto, FlowTableTargetDto } from './types'
import { writeFlows } from './utils/write-flows'

/** The persisted flow graph minus the table → target edges; targets are carried as sink detail. */
export type FlowGraphSummary = {
    tables: FlowTableDto[]
    sinks: FlowSinkDto[]
    targets: FlowSinkTargetDto[]
    tableSinkEdges: FlowTableSinkDto[]
}

/** The full persisted flow graph: the three node stores plus both edge stores. */
export type FlowGraphDetailed = FlowGraphSummary & {
    tableTargetEdges: FlowTableTargetDto[]
}

/**
 * Create the flow nodes and table → sink edges the backend configuration declares, so the
 * topology is complete before any traffic is observed. Existing rows are left untouched,
 * since seeding also runs on bootstraps that reuse a previous session's database.
 */
export const seedFlowGraph = (config: ConfigResponseBodyModel): Promise<void> =>
    writeFlows(async (resolve) => {
        for (const sink of config.sinks ?? []) {
            await resolve.sink(sink.type, sink.host)
        }

        for (const table of config.tables ?? []) {
            const tableId = await resolve.table(table.schema, table.name)

            for (const sink of table.sinks ?? []) {
                const sinkId = await resolve.sink(sink.type, sink.host)
                const existing = await db.flowConnectionTableSink
                    .where('[tableId+sinkId]')
                    .equals([tableId, sinkId])
                    .first()
                if (!existing) await db.flowConnectionTableSink.add({ tableId, sinkId, operationCount: 0 })
            }
        }
    })

/** Remove every stored flow node and edge. */
export const clearFlows = (): Promise<void> =>
    db.transaction('rw', FLOW_STORES(), async () => {
        await Promise.all([
            db.flowTables.clear(),
            db.flowSinks.clear(),
            db.flowSinkTargets.clear(),
            db.flowConnectionTableSink.clear(),
            db.flowConnectionTableTarget.clear(),
        ])
    })

/** Read the flow graph without the table → target edges, in one consistent snapshot. */
export const getFlowGraphSummary = (): Promise<FlowGraphSummary> =>
    db.transaction('r', [db.flowTables, db.flowSinks, db.flowSinkTargets, db.flowConnectionTableSink], async () => {
        const [tables, sinks, targets, tableSinkEdges] = await Promise.all([
            db.flowTables.toArray(),
            db.flowSinks.toArray(),
            db.flowSinkTargets.toArray(),
            db.flowConnectionTableSink.toArray(),
        ])
        return { tables, sinks, targets, tableSinkEdges }
    })

/** Read the whole flow graph, table → target edges included, in one consistent snapshot. */
export const getFlowGraphDetailed = (): Promise<FlowGraphDetailed> =>
    db.transaction('r', FLOW_STORES(), async () => {
        const [tables, sinks, targets, tableSinkEdges, tableTargetEdges] = await Promise.all([
            db.flowTables.toArray(),
            db.flowSinks.toArray(),
            db.flowSinkTargets.toArray(),
            db.flowConnectionTableSink.toArray(),
            db.flowConnectionTableTarget.toArray(),
        ])
        return { tables, sinks, targets, tableSinkEdges, tableTargetEdges }
    })
