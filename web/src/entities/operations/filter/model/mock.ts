import type { MultiComboBoxItem } from '@/shared/ui/multi-combo-box'

/** Destination sinks operations are delivered to. */
export const sinkItems: MultiComboBoxItem[] = [
    { value: 'rabbitmq1', label: 'rabbitmq1' },
    { value: 'kafka1', label: 'kafka1' },
    { value: 'http1', label: 'http1' },
    { value: 'nats-core', label: 'nats-core' },
    { value: 'rmq-stream', label: 'rmq-stream' },
]

/** Source tables operations originate from. */
export const tableItems: MultiComboBoxItem[] = [
    { value: 'public.users', label: 'public.users' },
    { value: 'public.posts', label: 'public.posts' },
    { value: 'public.events', label: 'public.events' },
    { value: 'public.orders', label: 'public.orders' },
    { value: 'public.sessions', label: 'public.sessions' },
    { value: 'public.comments', label: 'public.comments' },
    { value: 'public.payments', label: 'public.payments' },
]

/** Operation kinds (matches `Operation['op']`). */
export const operationItems: MultiComboBoxItem[] = [
    { value: 'insert', label: 'insert' },
    { value: 'update', label: 'update' },
    { value: 'delete', label: 'delete' },
    { value: 'select', label: 'select' },
]

/** Delivery statuses (matches `Operation['status']`). */
export const statusItems: MultiComboBoxItem[] = [
    { value: 'delivered', label: 'delivered' },
    { value: 'failed', label: 'failed' },
    { value: 'retrying', label: 'retrying' },
]
