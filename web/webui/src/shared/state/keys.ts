import type { MetricsRequestBodyModel, OperationsRequestBodyModel } from '@/shared/api'

/** Root of the metrics keys, kept apart so `keys` never references itself while initializing. */
const METRICS = ['metrics'] as const

/** Root of the operations keys. */
const OPERATIONS = ['operations'] as const

/** Operations request body without its window, which the block index stands for. */
export type OperationsFilterKey = Omit<OperationsRequestBodyModel, 'from' | 'to'>

export const keys = {
    queries: {
        metrics: (duration: MetricsRequestBodyModel['duration']) => [...METRICS, duration] as const,
        operations: (filter: OperationsFilterKey, block: number) => [...OPERATIONS, filter, block] as const,
        /** Newest deliveries matching `filter`, as the flows inspector lists them. */
        recentOperations: (filter: OperationsFilterKey) => [...OPERATIONS, 'recent', filter] as const,
    },
    /** Key prefixes, for invalidating every variant of a query at once. */
    prefixes: {
        metrics: () => METRICS,
        operations: () => OPERATIONS,
    },
    mutations: {},
} as const
