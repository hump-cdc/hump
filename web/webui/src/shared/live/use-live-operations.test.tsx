import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, test } from 'bun:test'

import { cleanup, renderHook } from '@testing-library/react'

import { fakeLiveStream, registerDom, unregisterDom } from './stream.fake'
import type { LiveOperationsBatch, LiveOperationsListener } from './types'
import { useLiveOperations } from './use-live-operations'

beforeAll(registerDom)
afterAll(unregisterDom)

beforeEach(() => fakeLiveStream.reset())
afterEach(cleanup)

const batch = [] as unknown as LiveOperationsBatch

describe('useLiveOperations', () => {
    test('hands a live batch to the handler', () => {
        const received: LiveOperationsBatch[] = []
        renderHook(() => useLiveOperations((next) => received.push(next)))

        fakeLiveStream.emit(batch)

        expect(received).toEqual([batch])
    })

    test('calls the latest handler without resubscribing', () => {
        const calls: string[] = []
        const first: LiveOperationsListener = () => calls.push('first')
        const second: LiveOperationsListener = () => calls.push('second')
        const { rerender } = renderHook(({ handler }) => useLiveOperations(handler), {
            initialProps: { handler: first },
        })

        rerender({ handler: second })
        fakeLiveStream.emit(batch)

        expect(calls).toEqual(['second'])
        expect(fakeLiveStream.listeners()).toBe(1)
    })

    test('unsubscribes on unmount', () => {
        const { unmount } = renderHook(() => useLiveOperations(() => {}))
        expect(fakeLiveStream.listeners()).toBe(1)

        unmount()

        expect(fakeLiveStream.listeners()).toBe(0)
    })
})
