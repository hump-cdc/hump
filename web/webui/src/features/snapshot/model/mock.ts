import type { SnapshotTable } from './types'

/** Tables offered on the snapshot page, in display order. */
export const SNAPSHOT_TABLES: SnapshotTable[] = [
    { id: 'users', name: 'public.users', totalRows: 1_284_300, sinks: ['kafka1', 'rabbitmq1'] },
    { id: 'orders', name: 'public.orders', totalRows: 915_000, sinks: ['kafka1', 'nats-core'] },
    { id: 'posts', name: 'public.posts', totalRows: 845_000, sinks: ['kafka1'] },
    { id: 'events', name: 'public.events', totalRows: 3_180_000, sinks: ['kafka1', 'nats-core', 'rabbitmq1'] },
    { id: 'sessions', name: 'public.sessions', totalRows: 805_000, sinks: ['nats-core'] },
    { id: 'comments', name: 'public.comments', totalRows: 612_400, sinks: ['kafka1'] },
    { id: 'payments', name: 'public.payments', totalRows: 248_900, sinks: ['kafka1', 'rabbitmq1'] },
    { id: 'audit_log', name: 'public.audit_log', totalRows: 1_904_000, sinks: ['rabbitmq1'] },
]

/** Pre-selected tables on first load (matches the "3 selected" design state). */
export const DEFAULT_SELECTED_IDS = ['users', 'orders', 'posts']
