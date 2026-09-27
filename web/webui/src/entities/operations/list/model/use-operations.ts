import {
    keepPreviousData,
    type QueryFunctionContext,
    useQueries,
    useQuery,
    useQueryClient,
} from '@tanstack/react-query'

import { useCallback, useEffect, useMemo, useState } from 'react'

import { useShallow } from 'zustand/react/shallow'

import { useOperationsFilter } from '@/entities/operations/filter/@x/list'
import { Operations as OperationsApi, type OperationsResponseBodyModel } from '@/shared/api'
import { useLiveOperations } from '@/shared/live'
import { keys, type OperationsFilterKey } from '@/shared/state'
import type { DataTableRange } from '@/shared/ui/data-table'
import { useDebounce } from '@/shared/utils'

import { BLOCK_SIZE, type BlockPage, blocksForRange, clampBlocks, pickTotal, rowAt, sameBlocks } from './blocks'
import { mapOperation } from './map-operation'
import { toOperationsFilter } from './to-request'
import type { Operation } from './types'

/** Quiet period (ms) after typing before the search is sent. */
const SEARCH_DEBOUNCE_MS = 300

/** Quiet period (ms) after a live batch before the loaded blocks refetch. */
const LIVE_REFETCH_DEBOUNCE_MS = 300

/** Longest (ms) a steady live stream may defer the refetch; batches arrive every 500 ms. */
const LIVE_REFETCH_MAX_WAIT_MS = 1_000

/** How long (ms) a block stays fresh when scrolled back to; live batches invalidate it sooner. */
const BLOCK_STALE_TIME_MS = 5_000

/** A block's query key. */
type BlockKey = ReturnType<typeof keys.queries.operations>

/**
 * Operations log for {@link OperationsList}, loaded from `/api/operations` in
 * blocks of {@link BLOCK_SIZE}: block 0 always, plus the blocks under the table's
 * viewport. Every response carries the matching total, which sizes the table.
 */
export const useOperations = () => {
    const filter = useRequestFilter()
    const [blocks, setBlocks] = useState<number[]>([0])

    const onVisibleRangeChange = useCallback(({ start, end }: DataTableRange) => {
        const next = blocksForRange(start, end, BLOCK_SIZE)
        setBlocks((previous) => (sameBlocks(previous, next) ? previous : next))
    }, [])

    // Block 0 keeps showing the previous filter's rows until the new ones arrive.
    const head = useQuery({
        ...blockQuery(filter, 0),
        placeholderData: keepPreviousData,
    })
    const headPage = head.data
    const rest = useQueries({
        queries: clampBlocks(blocks, headPage?.total ?? 0, BLOCK_SIZE)
            .filter((block) => block > 0)
            .map((block) => blockQuery(filter, block)),
        combine: combinePages,
    })

    const pages = useMemo(() => {
        const map = new Map<number, BlockPage<Operation>>()
        for (const page of [headPage, ...rest]) {
            if (page) {
                map.set(Math.floor(page.from / BLOCK_SIZE), page)
            }
        }
        return map
    }, [headPage, rest])
    const getRow = useCallback((index: number) => rowAt(pages, index, BLOCK_SIZE), [pages])

    useLiveRefetch()

    return {
        rowCount: filter ? (pickTotal([...pages.values()]) ?? 0) : 0,
        getRow,
        onVisibleRangeChange,
        isLoading: filter !== undefined && head.isPending,
    } as const
}

/** The store's filter as a request filter, with the free-text search debounced. */
const useRequestFilter = (): OperationsFilterKey | undefined => {
    const state = useOperationsFilter(
        useShallow((store) => ({
            sinks: store.sinks,
            tables: store.tables,
            operations: store.operations,
            statuses: store.statuses,
            sinksHydrated: store.sinksHydrated,
            tablesHydrated: store.tablesHydrated,
        })),
    )
    const search = useOperationsFilter((store) => store.search)
    const [text, setText] = useState(search)
    const commitText = useDebounce(setText, SEARCH_DEBOUNCE_MS)

    useEffect(() => {
        commitText(search)
    }, [search, commitText])

    return useMemo(() => toOperationsFilter(state, text), [state, text])
}

/** Refetches the loaded blocks after live deliveries, coalescing bursts. */
const useLiveRefetch = (): void => {
    const queryClient = useQueryClient()
    const refetch = useDebounce(
        () => void queryClient.invalidateQueries({ queryKey: keys.prefixes.operations() }, { cancelRefetch: false }),
        LIVE_REFETCH_DEBOUNCE_MS,
        { maxWait: LIVE_REFETCH_MAX_WAIT_MS },
    )

    useLiveOperations(refetch)
}

/** Query options of one block; disabled while the filter matches nothing. */
const blockQuery = (filter: OperationsFilterKey | undefined, block: number) => ({
    queryKey: keys.queries.operations(filter ?? {}, block),
    queryFn: fetchBlock,
    select: toPage,
    enabled: filter !== undefined,
    staleTime: BLOCK_STALE_TIME_MS,
})

/** Fetches the block its key names. */
const fetchBlock = async ({ queryKey, signal }: QueryFunctionContext<BlockKey>) => {
    const [, filter, block] = queryKey
    const { data } = await OperationsApi.list({
        body: { ...filter, from: block * BLOCK_SIZE, to: (block + 1) * BLOCK_SIZE },
        signal,
        throwOnError: true,
    })
    return { ...data, receivedAt: Date.now() }
}

/** Maps a response to list rows; stable, so TanStack Query memoizes the result per block. */
const toPage = (data: OperationsResponseBodyModel & { receivedAt: number }): BlockPage<Operation> => ({
    from: Number(data.from),
    total: Number(data.total),
    rows: (data.items ?? []).map(mapOperation),
    updatedAt: data.receivedAt,
})

/** Loaded pages of the non-head blocks; stable, so the combined array is memoized. */
const combinePages = (results: { data: BlockPage<Operation> | undefined }[]): (BlockPage<Operation> | undefined)[] =>
    results.map((result) => result.data)
