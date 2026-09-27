import { describe, expect, test } from 'bun:test'

import { BACKOFF_BASE_MS, BACKOFF_MAX_MS, createBackoff } from './backoff'

const take = (count: number, base?: number, max?: number): number[] => {
    const backoff = createBackoff(base, max)
    return Array.from({ length: count }, () => backoff.next())
}

describe('createBackoff', () => {
    test('doubles the delay on every attempt', () => {
        expect(take(5, 1_000, 30_000)).toEqual([1_000, 2_000, 4_000, 8_000, 16_000])
    })

    test('caps the delay at max', () => {
        expect(take(8, 1_000, 30_000).slice(-3)).toEqual([30_000, 30_000, 30_000])
    })

    test('defaults to the exported base and cap', () => {
        const backoff = createBackoff()
        expect(backoff.next()).toBe(BACKOFF_BASE_MS)

        for (let i = 0; i < 10; i += 1) {
            backoff.next()
        }

        expect(backoff.next()).toBe(BACKOFF_MAX_MS)
    })

    test('reset returns to the base delay', () => {
        const backoff = createBackoff(1_000, 30_000)
        backoff.next()
        backoff.next()
        backoff.reset()

        expect(backoff.next()).toBe(1_000)
    })
})
