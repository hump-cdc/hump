import { afterEach, beforeEach, describe, expect, jest, test } from 'bun:test'

import { debounce } from './debounce'

beforeEach(() => {
    jest.useFakeTimers()
})

afterEach(() => {
    jest.useRealTimers()
})

describe('debounce', () => {
    test('runs once after the quiet period, with the last arguments', () => {
        const calls: number[] = []
        const debounced = debounce((value: number) => calls.push(value), 300)

        debounced(1)
        jest.advanceTimersByTime(200)
        debounced(2)
        jest.advanceTimersByTime(299)
        expect(calls).toEqual([])

        jest.advanceTimersByTime(1)
        expect(calls).toEqual([2])
    })

    test('runs at least every maxWait under a steady stream of calls', () => {
        let runs = 0
        const debounced = debounce(() => runs++, 300, { maxWait: 1000 })

        for (let elapsed = 0; elapsed < 2000; elapsed += 250) {
            debounced()
            jest.advanceTimersByTime(250)
        }

        expect(runs).toBe(2)
    })

    test('counts maxWait from the first call of each burst', () => {
        const calls: number[] = []
        const debounced = debounce((value: number) => calls.push(value), 300, { maxWait: 500 })

        debounced(1)
        jest.advanceTimersByTime(300)
        expect(calls).toEqual([1])

        jest.advanceTimersByTime(1000)
        debounced(2)
        jest.advanceTimersByTime(299)
        expect(calls).toEqual([1])

        jest.advanceTimersByTime(1)
        expect(calls).toEqual([1, 2])
    })

    test('cancel drops the pending call', () => {
        let runs = 0
        const debounced = debounce(() => runs++, 300)

        debounced()
        debounced.cancel()
        jest.advanceTimersByTime(1000)

        expect(runs).toBe(0)
    })

    test('cancel starts a fresh maxWait window for the next burst', () => {
        let runs = 0
        const debounced = debounce(() => runs++, 300, { maxWait: 400 })

        debounced()
        jest.advanceTimersByTime(200)
        debounced()
        jest.advanceTimersByTime(100)
        debounced.cancel()

        debounced()
        jest.advanceTimersByTime(299)
        expect(runs).toBe(0)

        jest.advanceTimersByTime(1)
        expect(runs).toBe(1)
    })
})
