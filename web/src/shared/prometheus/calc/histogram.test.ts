import { describe, expect, test } from 'bun:test'

import type { HistogramBucket } from '../types'
import { mergeBuckets, parseBound, quantile } from './histogram'

/** Cumulative buckets from `[bound, count]` pairs, as a scrape reports them. */
const buckets = (...pairs: [number, number][]): HistogramBucket[] => pairs.map(([le, count]) => ({ le, count }))

describe('parseBound', () => {
    test('reads a finite bound', () => {
        expect(parseBound('0.005')).toBe(0.005)
    })

    test('reads +Inf as Infinity so it sorts last', () => {
        expect(parseBound('+Inf')).toBe(Number.POSITIVE_INFINITY)
        expect([parseBound('+Inf'), parseBound('1')].sort((a, b) => a - b)).toEqual([1, Number.POSITIVE_INFINITY])
    })
})

describe('mergeBuckets', () => {
    test('copies `from` when `into` is empty, without aliasing it', () => {
        const from = buckets([1, 2])
        const merged = mergeBuckets([], from)

        expect(merged).toEqual(from)
        expect(merged[0]).not.toBe(from[0])
    })

    test('sums matching bounds', () => {
        expect(mergeBuckets(buckets([1, 2], [5, 4]), buckets([1, 3], [5, 6]))).toEqual(buckets([1, 5], [5, 10]))
    })

    test('unions bounds the two lists do not share, in ascending order', () => {
        expect(mergeBuckets(buckets([5, 4]), buckets([1, 3], [10, 7]))).toEqual(buckets([1, 3], [5, 4], [10, 7]))
    })

    test('keeps +Inf last', () => {
        const merged = mergeBuckets(buckets([1, 1], [Number.POSITIVE_INFINITY, 9]), buckets([2, 3]))
        expect(merged.at(-1)?.le).toBe(Number.POSITIVE_INFINITY)
    })

    test('does not mutate its arguments', () => {
        const into = buckets([1, 2])
        mergeBuckets(into, buckets([1, 5]))
        expect(into).toEqual(buckets([1, 2]))
    })
})

describe('quantile', () => {
    test('interpolates linearly inside the bucket the rank lands in', () => {
        // 10 observations, 4 at or below 1, all 10 at or below 2. The median rank of 5 sits
        // one sixth of the way into the [1, 2] bucket.
        expect(quantile(buckets([1, 4], [2, 10]), 0.5)).toBeCloseTo(1 + (2 - 1) * (1 / 6), 10)
    })

    test('interpolates the first bucket from a lower bound of zero', () => {
        expect(quantile(buckets([2, 10]), 0.5)).toBe(1)
    })

    test('returns the bucket bound when the rank sits exactly on it', () => {
        expect(quantile(buckets([1, 5], [2, 10]), 0.5)).toBe(1)
    })

    test('returns the lower bound for an empty bucket the rank lands in', () => {
        expect(quantile(buckets([1, 5], [2, 5], [4, 10]), 0.5)).toBe(1)
    })

    test('is undefined when the distribution is empty', () => {
        expect(quantile([], 0.95)).toBeUndefined()
        expect(quantile(buckets([1, 0], [Number.POSITIVE_INFINITY, 0]), 0.95)).toBeUndefined()
    })

    test('is undefined when the quantile lands in the unbounded +Inf bucket', () => {
        expect(quantile(buckets([1, 5], [Number.POSITIVE_INFINITY, 10]), 0.95)).toBeUndefined()
    })

    test('reads the total off the last bucket, not the first', () => {
        expect(quantile(buckets([1, 2], [2, 4], [4, 100]), 0.5)).toBeGreaterThan(2)
    })
})
