import { useQuery, useQueryClient } from '@tanstack/react-query'

import { useMemo } from 'react'

import { type OperationItemModel, Operations as OperationsApi } from '@/shared/api'
import { useLiveOperations } from '@/shared/live'
import { keys } from '@/shared/state'
import { useDebounce } from '@/shared/utils'

import { toRecentFilter } from './recent-filter'
import type { SelectedItem } from './types'

/** How many of the newest deliveries the inspector lists. */
const RECENT_LIMIT = 5

/** Quiet period (ms) after a live batch before the list refetches. */
const LIVE_REFETCH_DEBOUNCE_MS = 300

/** Longest (ms) a steady live stream may defer the refetch; batches arrive every 500 ms. */
const LIVE_REFETCH_MAX_WAIT_MS = 1_000

/** One row of the recent operations list. */
type RecentOperation = {
    id: string
    op: string
    time: Date
    isSuccess: boolean
}

/** Newest deliveries of the selected node or edge from `/api/operations`, refetched on live deliveries. */
export const useRecentOperations = (selected: SelectedItem) => {
    const filter = useMemo(() => toRecentFilter(selected), [selected])
    const queryKey = keys.queries.recentOperations(filter)

    const query = useQuery({
        queryKey,
        queryFn: async ({ signal }) => {
            const { data } = await OperationsApi.list({
                body: { ...filter, from: 0, to: RECENT_LIMIT },
                signal,
                throwOnError: true,
            })
            return data
        },
        select: (data) => (data.items ?? []).map(toRecentOperation),
    })

    const queryClient = useQueryClient()
    const refetch = useDebounce(
        () => void queryClient.invalidateQueries({ queryKey }, { cancelRefetch: false }),
        LIVE_REFETCH_DEBOUNCE_MS,
        { maxWait: LIVE_REFETCH_MAX_WAIT_MS },
    )
    useLiveOperations(refetch)

    return { operations: query.data ?? [], isLoading: query.isPending } as const
}

/** Converts one API record to a list row. */
const toRecentOperation = (item: OperationItemModel): RecentOperation => ({
    id: String(item.seq),
    op: item.operation.toUpperCase(),
    time: new Date(item.startedAt),
    isSuccess: item.status === 'success',
})
