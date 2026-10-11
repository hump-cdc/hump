import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, spyOn, test } from 'bun:test'

import { cleanup, render } from '@testing-library/react'

import { LIVE_EVENT_QUERIES, LIVE_RECONNECT_QUERIES } from '@/shared/state'

import { LiveUpdates } from './live-updates'
import { fakeLiveStream, registerDom, unregisterDom } from './stream.fake'
import type { LiveOperationsBatch } from './types'

beforeAll(registerDom)
afterAll(unregisterDom)

beforeEach(() => fakeLiveStream.reset())
afterEach(cleanup)

const batch = [] as unknown as LiveOperationsBatch

/** Mounts `LiveUpdates` under a fresh client and records what it invalidates. */
const mount = () => {
    const client = new QueryClient()
    const invalidate = spyOn(client, 'invalidateQueries')
    const view = render(
        <QueryClientProvider client={client}>
            <LiveUpdates />
        </QueryClientProvider>,
    )

    return { invalidate, unmount: view.unmount }
}

describe('LiveUpdates', () => {
    test('holds the connection while mounted', () => {
        const { unmount } = mount()
        expect(fakeLiveStream.holds()).toBe(1)

        unmount()

        expect(fakeLiveStream.holds()).toBe(0)
    })

    test('invalidates the live-event queries on a batch', () => {
        const { invalidate } = mount()

        fakeLiveStream.emit(batch)

        expect(invalidate.mock.calls.map(([filter]) => filter)).toEqual(LIVE_EVENT_QUERIES)
    })

    test('invalidates the reconnect queries after a gap in the stream', () => {
        const { invalidate } = mount()

        fakeLiveStream.reconnect()

        expect(invalidate.mock.calls.map(([filter]) => filter)).toEqual(LIVE_RECONNECT_QUERIES)
    })
})
