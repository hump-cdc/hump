import { useQuery } from '@tanstack/react-query'

import { getApiConfigOptions } from '@/shared/api'

export const useConfig = () =>
    useQuery({
        ...getApiConfigOptions(),
        staleTime: Number.POSITIVE_INFINITY,
        gcTime: Number.POSITIVE_INFINITY,
    })
