import { useLayoutEffect } from 'react'

import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, mock, test } from 'bun:test'

import { act, cleanup, fireEvent } from '@testing-library/react'

import type { Operation } from '@/entities/operations/list/@x/details'
import { stubMatchMedia } from '@/shared/testing/browser-stubs.fake'
import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { flush, renderWithProviders } from '@/shared/testing/render.fake'

import { useOperationDetailsStore as useListDetailsStore } from '../@x/list'
import { OperationDetails, useOperationDetailsStore } from '../index'

// Radix and Base UI pick their layout effect when first imported, as a no-op without a DOM; bun shares
// modules across test files, so point them at React's own one whatever order the files load in.
mock.module('@radix-ui/react-use-layout-effect', () => ({ useLayoutEffect }))
mock.module('@base-ui/utils/useIsoLayoutEffect', () => ({ useIsoLayoutEffect: useLayoutEffect }))

/** A delivery that went through on the first attempt, with headers and a body. */
const delivered: Operation = {
    id: '1',
    time: new Date('2026-01-01T12:00:00Z'),
    table: 'public.users',
    op: 'insert',
    sink: 'kafka',
    sinkHost: 'kafka:9092',
    target: 'users',
    status: 'delivered',
    attempts: 1,
    durationMs: 12,
    payload: '{"id":1}',
    payloadSize: 8,
    metadata: { 'header.content-type': 'application/json', partition: '3' },
}

/** A delivery that needed a retry, without headers or a body. */
const retried: Operation = { ...delivered, id: '2', attempts: 2, payload: '', metadata: {} }

/** A delivery that failed for good. */
const failed: Operation = { ...delivered, id: '3', status: 'failed', error: 'connection refused' }

beforeAll(registerDom)
afterAll(unregisterDom)

beforeEach(() => useOperationDetailsStore.setState({ selected: null }))
afterEach(cleanup)

/** Selects `operation` the way the list does. */
const select = (operation: Operation): void => {
    act(() => useListDetailsStore.getState().select(operation))
}

describe('OperationDetails on desktop', () => {
    test('stays closed until an operation is selected', () => {
        const { queryByText } = renderWithProviders(<OperationDetails />)

        expect(queryByText('Operation details')).toBeNull()
    })

    test('shows the event, route, headers and payload of the selection', async () => {
        const { getByText, getAllByText, findByText } = renderWithProviders(<OperationDetails />)
        select(failed)

        expect(await findByText('Operation details')).toBeDefined()
        expect(getByText('connection refused')).toBeDefined()
        expect(getByText('content-type')).toBeDefined()
        expect(getByText('partition')).toBeDefined()
        expect(getAllByText('kafka:9092').length).toBeGreaterThan(0)
        expect(getByText('Payload')).toBeDefined()
    })

    test('leaves out the headers and shows a placeholder for an empty payload', async () => {
        const { queryByText, findByText } = renderWithProviders(<OperationDetails />)
        select(retried)

        expect(await findByText('Operation details')).toBeDefined()
        expect(queryByText('Headers')).toBeNull()
        expect(queryByText('Copy payload')).toBeNull()
    })

    test('keeps the drawer open on an outside click and clears the selection on close', async () => {
        const { getByLabelText, findByText } = renderWithProviders(<OperationDetails />)
        select(delivered)
        expect(await findByText('Operation details')).toBeDefined()

        await flush()
        fireEvent.pointerDown(document.body)
        fireEvent.focusIn(document.body)
        expect(useOperationDetailsStore.getState().selected?.id).toBe('1')

        fireEvent.click(getByLabelText('Close details'))
        expect(useOperationDetailsStore.getState().selected).toBeNull()
    })
})

describe('OperationDetails on mobile', () => {
    let restoreMatchMedia: () => void

    beforeEach(() => {
        restoreMatchMedia = stubMatchMedia(true)
    })
    afterEach(() => restoreMatchMedia())

    test('titles the sheet with the route and splits the body into tabs', async () => {
        const { findAllByRole, getAllByText } = renderWithProviders(<OperationDetails />)
        select(delivered)

        const tabs = await findAllByRole('tab')
        expect(tabs.map((tab) => tab.textContent)).toEqual(['Event', 'Details', 'Headers', 'Payload'])
        expect(getAllByText('public.users').length).toBeGreaterThan(0)
        expect(getAllByText('1×')[0]?.className).toContain('text-ink-muted')
    })

    test('hides the headers tab without headers and tints the attempts', async () => {
        const { findAllByRole, getByText } = renderWithProviders(<OperationDetails />)
        select(retried)

        const tabs = await findAllByRole('tab')
        expect(tabs.map((tab) => tab.textContent)).toEqual(['Event', 'Details', 'Payload'])
        expect(getByText('2×').className).toContain('text-amber')

        select(failed)
        expect(getByText('1×').className).toContain('text-danger')
        expect(getByText('connection refused')).toBeDefined()
    })
})
