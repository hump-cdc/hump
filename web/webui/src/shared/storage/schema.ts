import { db } from './db'

/**
 * Declare the IndexedDB schema, once on the main thread before the database is opened. Other
 * contexts open it in Dexie's dynamic mode and declare nothing.
 */
export const schema = (): void => {
    db.version(1).stores({
        meta: 'key',
        operations: 'id, timestamp, schema, table, operation, sinkType, host, target, success, durationMs',
        flow_tables: '++id, &[schema+table], schema, table',
        flow_sinks: '++id, &[sinkType+host], sinkType, host',
        flow_sink_targets: '++id, &[sinkId+target], sinkId, target',
        flow_connection_table_sink: '++id, &[tableId+sinkId], tableId, sinkId',
        flow_connection_table_target: '++id, &[tableId+targetId], tableId, targetId',
        metric_scalar_1m: '[metric+timestamp], timestamp',
        metric_scalar_5m: '[metric+timestamp], timestamp',
        metric_scalar_10m: '[metric+timestamp], timestamp',
        metric_sink_delivery_1m: '[sink+timestamp], timestamp',
        metric_sink_delivery_5m: '[sink+timestamp], timestamp',
        metric_sink_delivery_10m: '[sink+timestamp], timestamp',
        metric_sink_latency_1m: '[sink+timestamp], timestamp',
        metric_sink_latency_5m: '[sink+timestamp], timestamp',
        metric_sink_latency_10m: '[sink+timestamp], timestamp',
    })
}
