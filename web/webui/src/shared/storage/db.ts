import Dexie, { type Table } from 'dexie'

import type { FlowSinkDto, FlowSinkTargetDto, FlowTableDto, FlowTableSinkDto, FlowTableTargetDto } from './flows/types'
import type { MetaDto } from './meta/types'
import type { ScalarMetricDto, ScalarMetricName, SinkMetricDto } from './metrics/types'
import type { OperationDto } from './operations/types'

export class HumpDatabase extends Dexie {
    constructor() {
        super('hump')
    }

    /** Key/value metadata store (e.g. the persisted config hash); primary key is `key`. */
    get meta(): Table<MetaDto, string> {
        return this.table('meta')
    }

    /** CDC operations store; primary key is the operation `id`. */
    get operations(): Table<OperationDto, string> {
        return this.table('operations')
    }

    /** Scalar metric samples bucketed to 1-minute windows; primary key is `[metric+timestamp]`. */
    get metricScalar1m(): Table<ScalarMetricDto, [ScalarMetricName, number]> {
        return this.table('metric_scalar_1m')
    }

    /** Scalar metric samples bucketed to 5-minute windows; primary key is `[metric+timestamp]`. */
    get metricScalar5m(): Table<ScalarMetricDto, [ScalarMetricName, number]> {
        return this.table('metric_scalar_5m')
    }

    /** Scalar metric samples bucketed to 10-minute windows; primary key is `[metric+timestamp]`. */
    get metricScalar10m(): Table<ScalarMetricDto, [ScalarMetricName, number]> {
        return this.table('metric_scalar_10m')
    }

    /** Per-sink delivery samples bucketed to 1-minute windows; primary key is `[sink+timestamp]`. */
    get metricSinkDelivery1m(): Table<SinkMetricDto, [string, number]> {
        return this.table('metric_sink_delivery_1m')
    }

    /** Per-sink delivery samples bucketed to 5-minute windows; primary key is `[sink+timestamp]`. */
    get metricSinkDelivery5m(): Table<SinkMetricDto, [string, number]> {
        return this.table('metric_sink_delivery_5m')
    }

    /** Per-sink delivery samples bucketed to 10-minute windows; primary key is `[sink+timestamp]`. */
    get metricSinkDelivery10m(): Table<SinkMetricDto, [string, number]> {
        return this.table('metric_sink_delivery_10m')
    }

    /** Per-sink latency samples bucketed to 1-minute windows; primary key is `[sink+timestamp]`. */
    get metricSinkLatency1m(): Table<SinkMetricDto, [string, number]> {
        return this.table('metric_sink_latency_1m')
    }

    /** Per-sink latency samples bucketed to 5-minute windows; primary key is `[sink+timestamp]`. */
    get metricSinkLatency5m(): Table<SinkMetricDto, [string, number]> {
        return this.table('metric_sink_latency_5m')
    }

    /** Per-sink latency samples bucketed to 10-minute windows; primary key is `[sink+timestamp]`. */
    get metricSinkLatency10m(): Table<SinkMetricDto, [string, number]> {
        return this.table('metric_sink_latency_10m')
    }

    /** Per-table CDC counters; primary key is the autoincrement `id`. */
    get flowTables(): Table<FlowTableDto, number> {
        return this.table('flow_tables')
    }

    /** Per-sink delivery stats; primary key is the autoincrement `id`. */
    get flowSinks(): Table<FlowSinkDto, number> {
        return this.table('flow_sinks')
    }

    /** Per-target stats within a sink; primary key is the autoincrement `id`. */
    get flowSinkTargets(): Table<FlowSinkTargetDto, number> {
        return this.table('flow_sink_targets')
    }

    /** Table → sink edges; primary key is the autoincrement `id`. */
    get flowConnectionTableSink(): Table<FlowTableSinkDto, number> {
        return this.table('flow_connection_table_sink')
    }

    /** Table → sink-target edges; primary key is the autoincrement `id`. */
    get flowConnectionTableTarget(): Table<FlowTableTargetDto, number> {
        return this.table('flow_connection_table_target')
    }
}

/** App-wide singleton. */
export const db = new HumpDatabase()
