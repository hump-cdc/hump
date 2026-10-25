import type {
    ConfigResponseBodyModel,
    MetricsPointModel,
    MetricsResponseBodyModel,
    OperationItemModel,
    OperationsResponseBodyModel,
} from '@/shared/api'

/** Start of the newest metrics bucket the fixtures report. */
const METRICS_NOW = Date.UTC(2026, 0, 1, 12, 30)

/** A `GET /api/config` body: two tables and the sinks they deliver to. */
export const configResponse = (overrides: Partial<ConfigResponseBodyModel> = {}): ConfigResponseBodyModel => ({
    app: { version: '1.0.0' },
    configPath: '/etc/hump/hump.yaml',
    rawConfig: 'database:\n  host: localhost\n',
    database: { host: 'localhost', name: 'app', port: 5432, publication: 'hump', slot: 'hump' },
    sinks: [
        { type: 'kafka', host: 'kafka:9092' },
        { type: 'http', host: 'api.example.com' },
    ],
    tables: [
        {
            schema: 'public',
            name: 'users',
            replica: 'full',
            sinks: [{ type: 'kafka', host: 'kafka:9092' }],
            targets: [{ type: 'kafka', host: 'kafka:9092', target: 'users', operations: ['insert', 'update'] }],
        },
        { schema: 'public', name: 'orders', replica: 'full', sinks: null, targets: null },
    ],
    ...overrides,
})

/** A `POST /api/metrics` body: two buckets and a topology of one table delivered to two sinks. */
export const metricsResponse = (overrides: Partial<MetricsResponseBodyModel> = {}): MetricsResponseBodyModel => ({
    duration: '1m',
    bucketSeconds: 60,
    points: [
        metricsPoint(1, {
            delivered: 3,
            failed: 1,
            throughput: 1.234,
            replicationLagBytes: 512,
            memoryAllocBytes: 1_048_576,
            sinkDelivery: { kafka: 2 },
            sinkLatencyP95: { kafka: 1.234 },
        }),
        metricsPoint(0, {
            delivered: 4,
            failed: 0,
            throughput: 2,
            replicationLagBytes: 1024,
            memoryAllocBytes: 2_097_152,
            sinkDelivery: { kafka: 1, http: 3, nats: 1, rabbitmq: 1, redis: 1 },
            sinkLatencyP95: { http: 5 },
        }),
    ],
    tables: {
        'public.users': { i: { n: 3, p95: 4 }, u: { n: 2 }, d: { n: 1 }, t: { n: 0 } },
    },
    sinks: [
        {
            type: 'kafka',
            host: 'kafka:9092',
            operations: 5,
            success: 0.8,
            p95: 12,
            targets: [{ target: 'users', operations: 5, success: 0.8, p95: 12 }],
        },
        { type: 'http', host: 'api.example.com', operations: 0, targets: null },
    ],
    connections: [
        { table: 'public.users', sink: 'kafka', host: 'kafka:9092', target: 'users', success: 4, failure: 1 },
    ],
    ...overrides,
})

/** One `POST /api/operations` record; `seq` doubles as its id. */
export const operationItem = (seq: number, overrides: Partial<OperationItemModel> = {}): OperationItemModel => ({
    seq,
    database: 'app',
    table: 'public.users',
    operation: 'insert',
    sink: 'kafka',
    host: 'kafka:9092',
    target: 'users',
    status: 'success',
    startedAt: '2026-01-01T12:00:00Z',
    timestamp: '2026-01-01T12:00:00Z',
    duration: 12,
    retries: 0,
    attempts: null,
    payload: '{"id":1}',
    payloadSize: 8,
    metadata: {},
    ...overrides,
})

/** A `POST /api/operations` body holding `items`, starting at `from` out of `total`. */
export const operationsResponse = (
    items: OperationItemModel[],
    { from = 0, total = items.length }: { from?: number; total?: number } = {},
): OperationsResponseBodyModel => ({
    from,
    to: from + items.length,
    total,
    written: total,
    items,
})

/** One metrics bucket starting `minutesAgo` before `METRICS_NOW`. */
const metricsPoint = (minutesAgo: number, overrides: Partial<MetricsPointModel> = {}): MetricsPointModel => ({
    timestamp: new Date(METRICS_NOW - minutesAgo * 60_000).toISOString(),
    delivered: 0,
    failed: 0,
    throughput: 0,
    replicationLagBytes: 0,
    memoryAllocBytes: 0,
    sinkDelivery: {},
    sinkLatencyP95: {},
    ...overrides,
})
