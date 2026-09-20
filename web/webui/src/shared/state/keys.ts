import type { MetricsRequestBodyModel } from '@/shared/api'

/** Root of the metrics keys, kept apart so `keys` never references itself while initializing. */
const METRICS = ['metrics'] as const

export const keys = {
    queries: {
        metrics: (duration: MetricsRequestBodyModel['duration']) => [...METRICS, duration] as const,
    },
    /** Key prefixes, for invalidating every variant of a query at once. */
    prefixes: {
        metrics: () => METRICS,
    },
    mutations: {},
} as const
