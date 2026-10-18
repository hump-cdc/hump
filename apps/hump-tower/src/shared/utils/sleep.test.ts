import { afterEach, beforeEach, describe, expect, jest, test } from 'bun:test'

import { sleep } from './sleep'

beforeEach(() => {
    jest.useFakeTimers()
})

afterEach(() => {
    jest.useRealTimers()
})

describe('sleep', () => {
    test('resolves once the delay has passed', async () => {
        let resolved = false
        const pending = sleep(300).then(() => {
            resolved = true
        })

        jest.advanceTimersByTime(299)
        await Promise.resolve()
        expect(resolved).toBe(false)

        jest.advanceTimersByTime(1)
        await pending
        expect(resolved).toBe(true)
    })

    test('resolves as soon as the signal aborts', async () => {
        const controller = new AbortController()
        const pending = sleep(300, controller.signal)

        controller.abort()

        await pending
        expect(jest.getTimerCount()).toBe(0)
    })

    test('resolves at once for a signal that is already aborted', async () => {
        const controller = new AbortController()
        controller.abort()

        await sleep(300, controller.signal)

        expect(jest.getTimerCount()).toBe(0)
    })
})
