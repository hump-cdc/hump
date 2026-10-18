import type { MultiComboBoxItem } from '@/shared/ui/multi-combo-box'

/** Operation kinds (Postgres CDC). */
export const operationItems: MultiComboBoxItem[] = [
    { value: 'insert', label: 'insert' },
    { value: 'update', label: 'update' },
    { value: 'delete', label: 'delete' },
    { value: 'truncate', label: 'truncate' },
]

/** Delivery statuses (matches `Operation['status']`). */
export const statusItems: MultiComboBoxItem[] = [
    { value: 'delivered', label: 'delivered' },
    { value: 'failed', label: 'failed' },
]
