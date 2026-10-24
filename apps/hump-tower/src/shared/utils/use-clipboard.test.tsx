import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { act, cleanup, renderHook, waitFor } from '@testing-library/react'

import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'

import { useClipboard } from './use-clipboard'

beforeAll(registerDom)
afterAll(unregisterDom)
afterEach(cleanup)

describe('useClipboard', () => {
    test('writes the text, flags it copied and clears the flag after the delay', async () => {
        const written: string[] = []
        stubClipboard(async (text) => {
            written.push(text)
        })
        const { result } = renderHook(() => useClipboard(10))

        await act(async () => result.current.copy('hello'))

        expect(written).toEqual(['hello'])
        expect(result.current.copied).toBe(true)
        await waitFor(() => expect(result.current.copied).toBe(false))
    })

    test('stays not copied when the clipboard rejects', async () => {
        stubClipboard(() => Promise.reject(new Error('denied')))
        const { result } = renderHook(() => useClipboard())

        await act(async () => result.current.copy('hello'))

        expect(result.current.copied).toBe(false)
    })
})

/** Replaces `navigator.clipboard` with one whose `writeText` is `writeText`. */
const stubClipboard = (writeText: (text: string) => Promise<void>): void => {
    Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true })
}
