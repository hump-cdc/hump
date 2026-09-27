import type { MetricsDelivery, MetricsTable, MetricsTopology } from '@/entities/metrics/@x/inspector'

import type { SelectedItem, SinkKey } from './types'

/** Delivery figures of a sink, target or edge as the inspector renders them. */
export type DeliveryStats = {
    operations: number
    /** Success rate (0–100), floored like the canvas so one failure never reads as 100%; absent without an operation. */
    successRate?: number
    /** Successful deliveries; absent without an operation. */
    success?: number
    /** Failed deliveries; absent without an operation. */
    failure?: number
    /** P95 write latency in milliseconds; absent without an estimate or for an edge. */
    p95?: number
}

/** The table's row operations, or `undefined` when the metrics no longer report it. */
export const resolveTable = (topology: MetricsTopology, name: string): MetricsTable | undefined =>
    topology.tables.find((table) => table.name === name)

/** Delivery figures of the selected sink, target or edge; `undefined` when the metrics no longer report it. */
export const resolveDelivery = (
    topology: MetricsTopology,
    selected: Exclude<SelectedItem, { kind: 'table' }>,
): DeliveryStats | undefined => {
    switch (selected.kind) {
        case 'sink': {
            const sink = findSink(topology, selected)
            return sink && toStats(sink)
        }
        case 'target': {
            const target = findSink(topology, selected)?.targets.find(({ target }) => target === selected.name)
            return target && toStats(target)
        }
        case 'edge':
            return resolveEdge(topology, selected)
    }
}

/** Total row operations of a table across every operation kind. */
export const sumTableOps = (table: MetricsTable): number =>
    table.insert.count + table.update.count + table.delete.count + table.truncate.count

/** The sink endpoint identified by `key`. */
const findSink = (topology: MetricsTopology, key: SinkKey) =>
    topology.sinks.find((sink) => sink.type === key.type && sink.host === key.host)

/**
 * Sums the connections of an edge: every target of the table → sink route for a summary edge, the
 * one target for a detailed edge.
 */
const resolveEdge = (
    topology: MetricsTopology,
    edge: Extract<SelectedItem, { kind: 'edge' }>,
): DeliveryStats | undefined => {
    const connections = topology.connections.filter(
        (connection) =>
            connection.table === edge.table &&
            connection.sink === edge.type &&
            connection.host === edge.host &&
            (edge.target === undefined || connection.target === edge.target),
    )
    if (connections.length === 0) return undefined

    const success = connections.reduce((sum, connection) => sum + connection.success, 0)
    const failure = connections.reduce((sum, connection) => sum + connection.failure, 0)
    const operations = success + failure
    if (operations === 0) return toStats({ operations })

    return {
        operations,
        successRate: Math.floor((success / operations) * 100),
        success,
        failure,
    }
}

/** Converts a delivery share into the rendered rate and the success and failure counts it implies. */
const toStats = ({ operations, success: share, p95 }: MetricsDelivery): DeliveryStats => {
    const failure = share === undefined ? undefined : Math.round(operations * (1 - share))

    return {
        operations,
        successRate: share === undefined ? undefined : Math.floor(share * 100),
        success: failure === undefined ? undefined : operations - failure,
        failure,
        p95,
    }
}
