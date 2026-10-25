import { useLayoutEffect } from 'react'

import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, mock, test } from 'bun:test'

import { act, cleanup, fireEvent, waitFor } from '@testing-library/react'

import { useOperationDetailsStore } from '@/entities/operations/details/@x/list'
import { fakeApi, json } from '@/shared/testing/api.fake'
import { stubLayout } from '@/shared/testing/browser-stubs.fake'
import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { operationItem, operationsResponse } from '@/shared/testing/fixtures.fake'
import { fakeLiveStream } from '@/shared/testing/live.fake'
import { renderWithProviders } from '@/shared/testing/render.fake'

import { OperationsList } from '../index'
// Type-only modules: loaded for their side effects so the coverage report lists them.
import '../@x/details'
import '../model/types'

// Radix and Base UI pick their layout effect when first imported, as a no-op without a DOM; bun shares
// modules across test files, so point them at React's own one whatever order the files load in.
mock.module('@radix-ui/react-use-layout-effect', () => ({ useLayoutEffect }))
mock.module('@base-ui/utils/useIsoLayoutEffect', () => ({ useIsoLayoutEffect: useLayoutEffect }))

/** Rows of the fake log: a clean delivery, a retried one and a failed one. */
const ITEMS = [
    operationItem(1),
    operationItem(2, {
        duration: 40,
        retries: 1,
        attempts: [
            { attempt: 1, startedAt: '2026-01-01T12:00:00Z', duration: 20, status: 'retry', error: 'timeout' },
            { attempt: 2, startedAt: '2026-01-01T12:00:01Z', duration: 20, status: 'success' },
        ],
    }),
    operationItem(3, { status: 'failure', error: 'connection refused', operation: 'delete' }),
]

let restoreLayout: () => void

beforeAll(() => {
    registerDom()
    restoreLayout = stubLayout(1200, 600)
})
afterAll(async () => {
    restoreLayout()
    await unregisterDom()
})

beforeEach(() => {
    fakeLiveStream.reset()
    useOperationDetailsStore.setState({ selected: null })
    fakeApi.serve({ 'POST /api/operations': () => json(operationsResponse(ITEMS)) })
})
afterEach(cleanup)

describe('OperationsList', () => {
    test('renders every column of the loaded rows', async () => {
        const { findAllByText, getByText, getAllByText } = renderWithProviders(<OperationsList />)

        expect(await findAllByText('1×')).toHaveLength(2)
        expect(getAllByText('connection refused').length).toBeGreaterThan(0)
        expect(getByText('2×')).toBeDefined()
    })

    test('selects a clicked row into the details store and clears it on a click outside', async () => {
        const { findAllByText, container } = renderWithProviders(<OperationsList />)

        const [first] = await findAllByText('1×')
        if (!first) throw new Error('No row rendered')
        fireEvent.click(first)
        expect(useOperationDetailsStore.getState().selected?.id).toBe('1')

        const body = container.querySelector('.overflow-auto')
        if (!body) throw new Error('No table body rendered')
        fireEvent.click(body)
        expect(useOperationDetailsStore.getState().selected).toBeNull()
    })

    test('expands a retried row into its attempts', async () => {
        const { findAllByLabelText, findByText, getByText } = renderWithProviders(<OperationsList />)

        const chevrons = await findAllByLabelText('Expand row')
        expect(chevrons).toHaveLength(1)
        await act(async () => fireEvent.click(chevrons[0] as HTMLElement))

        expect(await findByText('#1')).toBeDefined()
        expect(getByText('#2')).toBeDefined()
        expect(getByText('timeout')).toBeDefined()
        await waitFor(() => expect(getByText('Attempt')).toBeDefined())
    })
})
