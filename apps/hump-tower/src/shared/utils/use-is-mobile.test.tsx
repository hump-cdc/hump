import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { act, cleanup, renderHook } from '@testing-library/react'

import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'

import { useIsMobile } from './use-is-mobile'

/** A media query list whose `matches` the test flips, notifying its `change` listeners. */
type FakeMediaQuery = {
    matches: boolean
    queries: string[]
    listeners: Set<() => void>
    flip: (matches: boolean) => void
}

let originalMatchMedia: typeof window.matchMedia
let media: FakeMediaQuery

beforeAll(() => {
    registerDom()
    originalMatchMedia = window.matchMedia
})
afterAll(async () => {
    window.matchMedia = originalMatchMedia
    await unregisterDom()
})
afterEach(cleanup)

describe('useIsMobile', () => {
    test('reads the breakpoint query on mount', () => {
        stubMatchMedia(true)

        const { result } = renderHook(() => useIsMobile(600))

        expect(result.current).toBe(true)
        expect(media.queries).toContain('(max-width: 599px)')
    })

    test('follows the query and stops listening on unmount', () => {
        stubMatchMedia(false)
        const { result, unmount } = renderHook(() => useIsMobile())
        expect(result.current).toBe(false)
        expect(media.queries).toContain('(max-width: 1023px)')

        act(() => media.flip(true))
        expect(result.current).toBe(true)

        unmount()
        expect(media.listeners.size).toBe(0)
    })
})

/** Points `window.matchMedia` at one shared fake list starting at `matches`. */
const stubMatchMedia = (matches: boolean): void => {
    const listeners = new Set<() => void>()
    const queries: string[] = []
    const list = {
        get matches() {
            return media.matches
        },
        addEventListener: (_type: string, listener: () => void) => listeners.add(listener),
        removeEventListener: (_type: string, listener: () => void) => listeners.delete(listener),
    }
    media = {
        matches,
        queries,
        listeners,
        flip: (next) => {
            media.matches = next
            for (const listener of listeners) listener()
        },
    }
    Object.defineProperty(window, 'matchMedia', {
        configurable: true,
        writable: true,
        value: (query: string) => {
            queries.push(query)
            return list
        },
    })
}
