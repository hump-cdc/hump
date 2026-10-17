import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, test } from 'bun:test'

import { act, cleanup, renderHook } from '@testing-library/react'

import { fakeLiveStream, registerDom, unregisterDom } from './stream.fake'
import { useLiveStatus } from './use-live-status'

beforeAll(registerDom)
afterAll(unregisterDom)

beforeEach(() => fakeLiveStream.reset())
afterEach(cleanup)

describe('useLiveStatus', () => {
    test('returns the current status', () => {
        fakeLiveStream.setStatus('connecting')

        const { result } = renderHook(() => useLiveStatus())

        expect(result.current).toBe('connecting')
    })

    test('re-renders on a status transition', () => {
        const { result } = renderHook(() => useLiveStatus())
        expect(result.current).toBe('idle')

        act(() => fakeLiveStream.setStatus('live'))

        expect(result.current).toBe('live')
    })
})
