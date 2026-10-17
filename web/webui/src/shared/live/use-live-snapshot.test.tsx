import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, test } from 'bun:test'

import { act, cleanup, renderHook } from '@testing-library/react'

import { fakeLiveStream, registerDom, unregisterDom } from './stream.fake'
import type { LiveSnapshot } from './types'
import { useLiveSnapshot } from './use-live-snapshot'

beforeAll(registerDom)
afterAll(unregisterDom)

beforeEach(() => fakeLiveStream.reset())
afterEach(cleanup)

const processing: LiveSnapshot = { state: 'processing', progress: 40 }

describe('useLiveSnapshot', () => {
    test('is undefined until the feed reports a snapshot', () => {
        const { result } = renderHook(() => useLiveSnapshot())

        expect(result.current).toBeUndefined()
    })

    test('follows the snapshot the feed reports', () => {
        const { result } = renderHook(() => useLiveSnapshot())

        act(() => fakeLiveStream.setSnapshot(processing))

        expect(result.current).toEqual(processing)
    })
})
