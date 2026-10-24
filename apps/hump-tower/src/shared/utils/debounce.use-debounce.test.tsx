import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { cleanup, renderHook } from '@testing-library/react'

import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'

import { useDebounce } from './debounce'

beforeAll(registerDom)
afterAll(unregisterDom)
afterEach(cleanup)

describe('useDebounce', () => {
    test('keeps one identity across renders and calls the latest function', async () => {
        const calls: string[] = []
        const { result, rerender } = renderHook(
            ({ tag }: { tag: string }) => useDebounce((value: number) => calls.push(`${tag}${value}`), 5),
            { initialProps: { tag: 'a' } },
        )
        const first = result.current

        rerender({ tag: 'b' })
        result.current(1)
        await wait(20)

        expect(result.current).toBe(first)
        expect(calls).toEqual(['b1'])
    })

    test('cancels a pending call on unmount', async () => {
        const calls: number[] = []
        const { result, unmount } = renderHook(() => useDebounce((value: number) => calls.push(value), 5))

        result.current(1)
        unmount()
        await wait(20)

        expect(calls).toEqual([])
    })
})

/** Resolves after `ms` real milliseconds. */
const wait = (ms: number): Promise<void> => new Promise((resolve) => setTimeout(resolve, ms))
