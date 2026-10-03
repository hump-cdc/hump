import { queryOptions, experimental_streamedQuery as streamedQuery, useMutation, useQuery } from '@tanstack/react-query'

import {
    errorMessage,
    type GetApiSnapshotResponse,
    Snapshot as SnapshotApi,
    type SnapshotStartTableModel,
} from '@/shared/api'
import { keys } from '@/shared/state'
import { toast } from '@/shared/ui/toast'

import type { Sink, SnapshotState, Table } from './types'

const MAX_RETRY_DELAY_MS = 10_000

/**
 * Stream of the server's snapshot state: the whole state on connect and again on every change,
 * reopened whenever it drops. Every hook below reads it under one key, so they share one connection.
 */
const snapshotQuery = queryOptions({
    queryKey: keys.queries.snapshot(),
    queryFn: streamedQuery<GetApiSnapshotResponse, GetApiSnapshotResponse | undefined>({
        reducer: (_, event) => event,
        initialValue: undefined,
        refetchMode: 'append',
        streamFn: async function* ({ signal }) {
            const { stream } = await SnapshotApi.stream({
                signal,
                sseMaxRetryAttempts: 1,
                onSseError: (error) => {
                    if (!signal.aborted) console.warn('[snapshot] stream error', error)
                },
            })

            yield* stream

            if (!signal.aborted) throw new Error('Snapshot stream closed')
        },
    }),
    retry: true,
    retryDelay: (attempt) => Math.min(1_000 * 2 ** attempt, MAX_RETRY_DELAY_MS),
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnMount: 'always',
    refetchOnWindowFocus: 'always',
    refetchOnReconnect: false,
    select: (event): SnapshotState | undefined => {
        switch (event?.state) {
            case undefined:
                return undefined
            case 'idle':
                return { state: 'idle' }
            case 'processing':
                return {
                    state: 'processing',
                    startedAt: new Date(event.startedAt),
                    tables: (event.tables ?? []).map(
                        (table): Table => ({
                            id: `${table.schema}.${table.name}`,
                            schema: table.schema,
                            table: table.name,
                            name: `${table.schema}.${table.name}`,
                            where: table.where ?? '',
                            status: table.status,
                            rowsRead: Number(table.rows),
                            totalRows: Number(table.estimatedRows),
                            sinks: (table.sinks ?? []).map(
                                ({ type, host, target, count }): Sink => ({
                                    type,
                                    host,
                                    target,
                                    count: Number(count),
                                }),
                            ),
                        }),
                    ),
                }
            case 'done':
                return {
                    state: 'done',
                    startedAt: new Date(event.startedAt),
                    finishedAt: new Date(event.finishedAt),
                    tables: (event.tables ?? []).map(
                        (table): Table => ({
                            id: `${table.schema}.${table.name}`,
                            schema: table.schema,
                            table: table.name,
                            name: `${table.schema}.${table.name}`,
                            where: '',
                            status: table.status,
                            error: table.error || undefined,
                            rowsRead: Number(table.rows),
                            totalRows: 0,
                            sinks: (table.sinks ?? []).map(
                                ({ type, host, target, count }): Sink => ({
                                    type,
                                    host,
                                    target,
                                    count: Number(count),
                                }),
                            ),
                        }),
                    ),
                }
        }
    },
})

/** State of the server's snapshot run; re-renders on every state the server sends. */
export const useSnapshotState = () => useQuery(snapshotQuery)

/** Starts a snapshot of the given tables; the run itself arrives over the snapshot stream. */
export const useStartSnapshot = () =>
    useMutation({
        mutationKey: keys.mutations.snapshotStart(),
        mutationFn: async (tables: SnapshotStartTableModel[]) => {
            await SnapshotApi.start({ body: { tables }, throwOnError: true })
        },
        onError: (error) => toast.error('Failed to start the snapshot', { description: errorMessage(error) }),
    })

/**
 * Stops the snapshot in progress, or clears a finished one back to idle; the new state arrives over
 * the snapshot stream.
 */
export const useStopSnapshot = () =>
    useMutation({
        mutationKey: keys.mutations.snapshotStop(),
        mutationFn: async () => {
            await SnapshotApi.stop({ throwOnError: true })
        },
        onError: (error) => toast.error('Failed to stop the snapshot', { description: errorMessage(error) }),
    })
