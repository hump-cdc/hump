import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, type Mock, spyOn, test } from 'bun:test'

import { act, cleanup, waitFor } from '@testing-library/react'

import { operationItems, statusItems, useOperationsFilter } from '@/entities/operations/filter/@x/list'
import { type ApiHandler, fakeApi, json } from '@/shared/testing/api.fake'
import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { operationItem, operationsResponse } from '@/shared/testing/fixtures.fake'
import { fakeLiveStream } from '@/shared/testing/live.fake'
import { flush, renderHookWithProviders } from '@/shared/testing/render.fake'

import { useOperations } from './use-operations'

/** Matching operations the fake log holds. */
const TOTAL = 250

/** Answers a block of the log with one row per block, numbered by its index. */
const operations: ApiHandler = (_, body) => {
    const from = Number((body as { from: number }).from)
    return json(operationsResponse([operationItem(from + 1)], { from, total: TOTAL }))
}

/** Filter that allows every option, as the store starts. */
const ALLOW_ALL = {
    sinks: null,
    targets: null,
    tables: null,
    operations: operationItems.map(({ value }) => value),
    statuses: statusItems.map(({ value }) => value),
    search: '',
}

/** Request bodies the fake log received. */
const requests = (): unknown[] =>
    fakeApi
        .calls()
        .filter(({ route }) => route === 'POST /api/operations')
        .map(({ body }) => body)

beforeAll(registerDom)
afterAll(unregisterDom)

// The persist middleware warns when it cannot write; the warning is muted.
let warn: Mock<typeof console.warn>

beforeEach(() => {
    warn = spyOn(console, 'warn').mockImplementation(() => {})
    fakeLiveStream.reset()
    useOperationsFilter.setState(ALLOW_ALL)
    fakeApi.serve({ 'POST /api/operations': operations })
})

afterEach(() => {
    cleanup()
    warn.mockRestore()
})

describe('useOperations', () => {
    test('loads block 0 and sizes the table by its total', async () => {
        const { result } = renderHookWithProviders(() => useOperations())
        expect(result.current.isLoading).toBe(true)
        expect(result.current.rowCount).toBe(0)

        await waitFor(() => expect(result.current.rowCount).toBe(TOTAL))

        expect(result.current.isLoading).toBe(false)
        expect(result.current.getRow(0)?.id).toBe('1')
        expect(result.current.getRow(100)).toBeUndefined()
        expect(requests()).toEqual([{ from: 0, to: 100 }])
    })

    test('loads the blocks under the visible range, once per block', async () => {
        const { result } = renderHookWithProviders(() => useOperations())
        await waitFor(() => expect(result.current.rowCount).toBe(TOTAL))

        act(() => result.current.onVisibleRangeChange({ start: 150, end: 160 }))
        await waitFor(() => expect(result.current.getRow(100)?.id).toBe('101'))
        act(() => result.current.onVisibleRangeChange({ start: 120, end: 130 }))
        await flush()

        expect(requests()).toEqual([
            { from: 0, to: 100 },
            { from: 100, to: 200 },
        ])
    })

    test('sends nothing while a facet matches nothing', async () => {
        useOperationsFilter.setState({ statuses: [] })

        const { result } = renderHookWithProviders(() => useOperations())
        await flush()

        expect(result.current).toMatchObject({ rowCount: 0, isLoading: false })
        expect(requests()).toEqual([])
    })

    test('sends the search text once typing pauses', async () => {
        const { result } = renderHookWithProviders(() => useOperations())
        await waitFor(() => expect(result.current.rowCount).toBe(TOTAL))

        act(() => useOperationsFilter.getState().setSearch('boom'))
        await flush(350)

        expect(requests().at(-1)).toEqual({ from: 0, to: 100, text: 'boom' })
    })

    test('refetches the loaded blocks after live deliveries', async () => {
        const { result } = renderHookWithProviders(() => useOperations())
        await waitFor(() => expect(result.current.rowCount).toBe(TOTAL))

        act(() => fakeLiveStream.emit([]))
        await flush(350)

        expect(requests()).toHaveLength(2)
    })
})
