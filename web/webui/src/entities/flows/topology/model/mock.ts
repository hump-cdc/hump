import type { EdgeSpec, SinkSpec, TableSpec } from './types'

/** Source tables, rendered as the left column of the topology. */
export const tables: TableSpec[] = [
    { id: 't-users', name: 'public.users', rate: '842', trend: 'up' },
    { id: 't-posts', name: 'public.posts', rate: '210', trend: 'up' },
    { id: 't-events', name: 'public.events', rate: '37', trend: 'up' },
]

/** Destination sinks, rendered as the right column of the topology. */
export const sinks: SinkSpec[] = [
    { id: 's-kafka', name: 'kafka1', subtitle: 'kafka', rate: '842/s', health: 'ok', successRate: 100 },
    { id: 's-natsc', name: 'nats-core', subtitle: 'nats', rate: '840/s', health: 'ok', successRate: 99 },
    { id: 's-natsj', name: 'nats-jetstream', subtitle: 'jetstream', rate: '839/s', health: 'ok', successRate: 80 },
    { id: 's-rabbit', name: 'rabbitmq1', subtitle: 'amqp', rate: '211/s', health: 'error', successRate: 25 },
    { id: 's-webhook', name: 'webhook-http', subtitle: 'http', rate: '588/s', health: 'ok', successRate: 99 },
]

/** Weighted edge: `opacity`/`strokeWidth` encode relative throughput of the route. */
const edge = (source: string, target: string, opacity: number, strokeWidth: number): EdgeSpec => ({
    id: `${source}-${target}`,
    source,
    target,
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
