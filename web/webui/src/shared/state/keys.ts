import type { MetricsRequestBodyModel } from '@/shared/api'

export const keys = {
    queries: {
        metrics: (duration: MetricsRequestBodyModel['duration']) => ['metrics', duration] as const,
    },
    mutations: {},
} as const
