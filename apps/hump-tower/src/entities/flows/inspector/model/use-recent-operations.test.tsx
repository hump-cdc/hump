import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, test } from 'bun:test'

import { cleanup, waitFor } from '@testing-library/react'

import { fakeApi, json } from '@/shared/testing/api.fake'
import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { operationItem, operationsResponse } from '@/shared/testing/fixtures.fake'
import { fakeLiveStream } from '@/shared/testing/live.fake'
import { flush, renderHookWithProviders } from '@/shared/testing/render.fake'

import type { SelectedItem } from './types'
import { useRecentOperations } from './use-recent-operations'

const TABLE: SelectedItem = { kind: 'table', id: 'public.users', name: 'public.users' }

/** Requests the hook sent to the operations endpoint. */
const operationCalls = () => fakeApi.calls().filter(({ route }) => route === 'POST /api/operations')

beforeAll(registerDom)
afterAll(unregisterDom)
beforeEach(() => fakeLiveStream.reset())
afterEach(cleanup)

describe('useRecentOperations', () => {
    test('lists the newest deliveries of the selection', async () => {
        fakeApi.serve({
            'POST /api/operations': () =>
                json(
                    operationsResponse([
                        operationItem(1),
                        operationItem(2, { status: 'failed', operation: 'update', startedAt: '2026-01-01T12:01:00Z' }),
                    ]),
                ),
        })

        const { result } = renderHookWithProviders(() => useRecentOperations(TABLE))

        expect(result.current).toEqual({ operations: [], isLoading: true })
        await waitFor(() => expect(result.current.isLoading).toBe(false))
        expect(result.current.operations).toEqual([
            { id: '1', op: 'INSERT', time: new Date('2026-01-01T12:00:00Z'), isSuccess: true },
            { id: '2', op: 'UPDATE', time: new Date('2026-01-01T12:01:00Z'), isSuccess: false },
        ])
        expect(operationCalls().map(({ body }) => body)).toEqual([{ tables: ['public.users'], from: 0, to: 5 }])
    })

    test('reads a missing item list as no deliveries', async () => {
        fakeApi.serve({ 'POST /api/operations': () => json({ ...operationsResponse([]), items: null }) })

        const { result } = renderHookWithProviders(() => useRecentOperations(TABLE))

        await waitFor(() => expect(result.current.isLoading).toBe(false))
        expect(result.current.operations).toEqual([])
    })

    test('refetches after a live batch', async () => {
        fakeApi.serve({ 'POST /api/operations': () => json(operationsResponse([operationItem(1)])) })

        const { result } = renderHookWithProviders(() => useRecentOperations(TABLE))
        await waitFor(() => expect(result.current.isLoading).toBe(false))

        fakeLiveStream.emit([])
        await flush(400)

        await waitFor(() => expect(operationCalls()).toHaveLength(2))
    })
})
