import type { MetricsSink, MetricsTopology } from '@/entities/metrics/@x/flows'

import type { DetailedSinkSpec, EdgeSpec, SinkSpec, TableSpec } from './types'

/** Everything both topology views render, derived from the metrics topology. */
export type FlowsTopology = {
    tables: TableSpec[]
    sinks: SinkSpec[]
    edges: EdgeSpec[]
    detailedSinks: DetailedSinkSpec[]
    detailedEdges: EdgeSpec[]
}

/** A success rate below this share renders the sink or target as failing. */
const HEALTH_ERROR_BELOW = 0.5

/** Edge weight range: the busiest route gets the maximum, an idle one the minimum. */
const EDGE_OPACITY = { min: 0.25, max: 1 }
const EDGE_STROKE_WIDTH = { min: 1.5, max: 5 }

/** Target kind label per sink type, matching what the sink renders its target to. */
const TARGET_KINDS: Record<string, string> = {
    kafka: 'topic',
    'nats-core': 'subject',
    'nats-jetstream': 'subject',
    rabbitmq: 'exchange',
    'rabbitmq-stream': 'stream',
    http: 'url',
}

/** Node id of a table: its schema-qualified name. */
export const tableId = (schema: string, table: string): string => `${schema}.${table}`

/** Node id of a sink, which is identified by its type and host; `|` never appears in either. */
export const sinkId = (type: string, host: string): string => `${type}|${host}`

/** Success share as a whole percentage, floored so a single failure never reads as 100%. */
const toSuccessRate = (success: number | undefined): number | undefined =>
    success === undefined ? undefined : Math.floor(success * 100)

/** Failing only when there is a success rate and it is below {@link HEALTH_ERROR_BELOW}. */
const toHealth = (success: number | undefined): 'ok' | 'error' =>
    success !== undefined && success < HEALTH_ERROR_BELOW ? 'error' : 'ok'

/** Edge scaled by its share of the busiest route's operations. */
const weightedEdge = (
    id: string,
    source: string,
    target: string,
    operations: number,
    max: number,
    targetHandle?: string,
): EdgeSpec => {
    const ratio = max > 0 ? operations / max : 0

    return {
        id,
        source,
        target,
        targetHandle,
        style: {
            opacity: EDGE_OPACITY.min + (EDGE_OPACITY.max - EDGE_OPACITY.min) * ratio,
            strokeWidth: EDGE_STROKE_WIDTH.min + (EDGE_STROKE_WIDTH.max - EDGE_STROKE_WIDTH.min) * ratio,
        },
    }
}

/** Summary sink card: one per sink endpoint, named after its type. */
const toSinkSpec = (sink: MetricsSink): SinkSpec => ({
    id: sinkId(sink.type, sink.host),
    name: sink.type,
    type: sink.type,
    operations: sink.operations,
    p95: sink.p95,
    successRate: toSuccessRate(sink.success),
    health: toHealth(sink.success),
})

/** Detailed sink card: the sink with its host and one row per target, the target string doubling as row id. */
const toDetailedSinkSpec = (sink: MetricsSink): DetailedSinkSpec => ({
    id: sinkId(sink.type, sink.host),
    name: sink.type,
    type: sink.type,
    host: sink.host,
    count: sink.operations,
    targets: sink.targets.map((target) => ({
        id: target.target,
        name: target.target,
        kind: TARGET_KINDS[sink.type] ?? 'target',
        count: target.operations,
        p95: target.p95,
        successRate: toSuccessRate(target.success),
        health: toHealth(target.success),
    })),
})

/**
 * Maps the metrics topology onto both views. Detailed edges run table → sink target as reported;
 * summary edges sum them per table → sink. Edge weights are relative to the busiest edge of each view.
 */
export const toFlowsTopology = ({ tables, sinks, connections }: MetricsTopology): FlowsTopology => {
    const summaryOps = new Map<string, { source: string; target: string; operations: number }>()
    for (const connection of connections) {
        const source = connection.table
        const target = sinkId(connection.sink, connection.host)
        const id = `${source}>${target}`
        const entry = summaryOps.get(id) ?? { source, target, operations: 0 }
        entry.operations += connection.operations
        summaryOps.set(id, entry)
    }

    const summaryMax = Math.max(0, ...[...summaryOps.values()].map(({ operations }) => operations))
    const detailedMax = Math.max(0, ...connections.map(({ operations }) => operations))

    return {
        tables: tables.map((table) => ({
            id: table.name,
            name: table.name,
            ops: {
                insert: { count: table.insert },
                update: { count: table.update },
                delete: { count: table.delete },
                truncate: { count: table.truncate },
            },
        })),
        sinks: sinks.map(toSinkSpec),
        edges: [...summaryOps].map(([id, { source, target, operations }]) =>
            weightedEdge(id, source, target, operations, summaryMax),
        ),
        detailedSinks: sinks.map(toDetailedSinkSpec),
        detailedEdges: connections.map((connection) => {
            const target = sinkId(connection.sink, connection.host)
            return weightedEdge(
                `${connection.table}>${target}>${connection.target}`,
                connection.table,
                target,
                connection.operations,
                detailedMax,
                connection.target,
            )
        }),
    }
}
