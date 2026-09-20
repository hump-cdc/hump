import type { DetailedSinkSpec, EdgeSpec, GhostSpec, SinkSpec, TableSpec } from './types'

/** Pre-formatted insert/update/delete/truncate counts for a table. */
const ops = (insert: string, update: string, del: string, truncate: string): TableSpec['ops'] => ({
    insert: { count: insert },
    update: { count: update },
    delete: { count: del },
    truncate: { count: truncate },
})

/** Source tables, rendered as the left column of the topology. */
export const tables: TableSpec[] = [
    { id: 't-users', name: 'public.users', ops: ops('842', '311', '47', '0') },
    { id: 't-posts', name: 'public.posts', ops: ops('210', '540', '88', '1') },
    { id: 't-events', name: 'public.events', ops: ops('37', '12', '4', '0') },
]

/** Destination sinks, rendered as the right column of the topology. */
export const sinks: SinkSpec[] = [
    { id: 's-kafka', name: 'kafka1', type: 'kafka', operations: '1.24M', p95: '18ms', successRate: 100, health: 'ok' },
    {
        id: 's-natsc',
        name: 'nats-core',
        type: 'nats-core',
        operations: '1.1M',
        p95: '4ms',
        successRate: 99,
        health: 'ok',
    },
    {
        id: 's-natsj',
        name: 'nats-jetstream',
        type: 'nats-jetstream',
        operations: '980K',
        p95: '12ms',
        successRate: 80,
        health: 'ok',
    },
    {
        id: 's-rabbit',
        name: 'rabbitmq1',
        type: 'rabbitmq',
        operations: '486K',
        p95: '31ms',
        successRate: 25,
        health: 'error',
    },
    {
        id: 's-webhook',
        name: 'webhook-http',
        type: 'http',
        operations: '712K',
        p95: '64ms',
        successRate: 99,
        health: 'ok',
    },
]

/**
 * Weighted edge: `opacity`/`strokeWidth` encode relative throughput of the route. `targetHandle`
 * addresses a target inside a detailed sink; omitted for summary sinks and ghosts.
 */
const edge = (
    source: string,
    target: string,
    opacity: number,
    strokeWidth: number,
    targetHandle?: string,
): EdgeSpec => ({
    id: [source, target, targetHandle].filter(Boolean).join('-'),
    source,
    target,
    targetHandle,
    style: { opacity, strokeWidth },
})

/** Table → sink routes connecting the two columns. */
export const edges: EdgeSpec[] = [
    edge('t-users', 's-kafka', 1, 5),
    edge('t-users', 's-natsc', 0.9, 3),
    edge('t-users', 's-natsj', 0.7, 2.5),
    edge('t-users', 's-rabbit', 0.5, 2.5),
    edge('t-users', 's-webhook', 0.55, 2),
    edge('t-posts', 's-kafka', 0.45, 2),
    edge('t-posts', 's-natsj', 0.6, 2),
    edge('t-events', 's-rabbit', 0.3, 2),
    edge('t-events', 's-webhook', 0.25, 1.5),
]

/** The same sinks as {@link sinks}, broken down into their delivery targets for the detailed view. */
export const detailedSinks: DetailedSinkSpec[] = [
    {
        id: 's-kafka',
        name: 'kafka1',
        type: 'kafka',
        host: 'localhost:9092',
        count: '1.24M',
        targets: [
            { id: 'users', name: 'users', kind: 'topic', count: '820K', p95: '16ms', successRate: 100, health: 'ok' },
            { id: 'posts', name: 'posts', kind: 'topic', count: '420K', p95: '21ms', successRate: 100, health: 'ok' },
        ],
    },
    {
        id: 's-natsc',
        name: 'nats-core',
        type: 'nats-core',
        host: 'localhost:4222',
        count: '1.1M',
        targets: [
            {
                id: 'users',
                name: 'hump.users',
                kind: 'subject',
                count: '1.1M',
                p95: '4ms',
                successRate: 99,
                health: 'ok',
            },
        ],
    },
    {
        id: 's-natsj',
        name: 'nats-jetstream',
        type: 'nats-jetstream',
        host: 'localhost:4222',
        count: '980K',
        targets: [
            { id: 'users', name: 'USERS', kind: 'stream', count: '610K', p95: '11ms', successRate: 90, health: 'ok' },
            { id: 'posts', name: 'POSTS', kind: 'stream', count: '370K', p95: '14ms', successRate: 64, health: 'ok' },
        ],
    },
    {
        id: 's-rabbit',
        name: 'rabbitmq1',
        type: 'rabbitmq',
        host: 'localhost:5672',
        count: '486K',
        targets: [
            { id: 'users', name: 'users', kind: 'queue', count: '300K', p95: '28ms', successRate: 30, health: 'error' },
            {
                id: 'events',
                name: 'events',
                kind: 'queue',
                count: '186K',
                p95: '36ms',
                successRate: 17,
                health: 'error',
            },
        ],
    },
    {
        id: 's-webhook',
        name: 'webhook-http',
        type: 'http',
        host: 'api.example.com',
        count: '712K',
        targets: [
            {
                id: 'users',
                name: '/hooks/users',
                kind: 'endpoint',
                count: '540K',
                p95: '60ms',
                successRate: 99,
                health: 'ok',
            },
            {
                id: 'events',
                name: '/hooks/events',
                kind: 'endpoint',
                count: '172K',
                p95: '71ms',
                successRate: 99,
                health: 'ok',
            },
        ],
    },
]

/** Placeholder sinks not configured yet, stacked below {@link detailedSinks}. */
export const ghosts: GhostSpec[] = [
    { id: 'g-clickhouse', type: 'clickhouse' },
    { id: 'g-s3', type: 's3' },
]

/** Table → sink target routes of the detailed view; the same routes as {@link edges} plus ghost links. */
export const detailedEdges: EdgeSpec[] = [
    edge('t-users', 's-kafka', 1, 5, 'users'),
    edge('t-users', 's-natsc', 0.9, 3, 'users'),
    edge('t-users', 's-natsj', 0.7, 2.5, 'users'),
    edge('t-users', 's-rabbit', 0.5, 2.5, 'users'),
    edge('t-users', 's-webhook', 0.55, 2, 'users'),
    edge('t-posts', 's-kafka', 0.45, 2, 'posts'),
    edge('t-posts', 's-natsj', 0.6, 2, 'posts'),
    edge('t-posts', 'g-s3', 0.2, 1.5),
    edge('t-events', 's-rabbit', 0.3, 2, 'events'),
    edge('t-events', 's-webhook', 0.25, 1.5, 'events'),
    edge('t-events', 'g-clickhouse', 0.2, 1.5),
]
