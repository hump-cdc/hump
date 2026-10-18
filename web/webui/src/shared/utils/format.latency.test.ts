import { describe, expect, test } from 'bun:test'

import { formatLatency, scaleLatency } from './format.latency'

describe('formatLatency', () => {
    test('prints whole milliseconds below one second', () => {
        expect(formatLatency(12.4)).toBe('12ms')
    })

    test('prints seconds with two fraction digits from one second on', () => {
        expect(formatLatency(1000)).toBe('1.00s')
        expect(formatLatency(1240)).toBe('1.24s')
    })

    test('compares against the boundary before rounding', () => {
        expect(formatLatency(999.6)).toBe('1000ms')
    })

    test('clamps a negative latency to zero', () => {
        expect(formatLatency(-5)).toBe('0ms')
    })
})

describe('scaleLatency', () => {
    test('keeps milliseconds below one second', () => {
        expect(scaleLatency(12.4)).toEqual({ value: 12, suffix: 'ms', fractionDigits: 0 })
    })

    test('switches to seconds from one second on', () => {
        expect(scaleLatency(1240)).toEqual({ value: 1.24, suffix: 's', fractionDigits: 2 })
    })

    test('clamps a negative latency to zero', () => {
        expect(scaleLatency(-5)).toEqual({ value: 0, suffix: 'ms', fractionDigits: 0 })
    })
})
