import { describe, expect, test } from 'bun:test'

import { formatNumber, formatNumber1, formatNumber2, formatNumberCompact } from './format.number'

/** The thousands separator of every `formatNumber*`: a non-breaking space. */
const NBSP = ' '

describe('formatNumber', () => {
    test('rounds to an integer grouped with non-breaking spaces', () => {
        expect(formatNumber(1284300.456)).toBe(`1${NBSP}284${NBSP}300`)
        expect(formatNumber(-1234.5)).toBe(`-1${NBSP}235`)
    })

    test('appends the suffix with no space', () => {
        expect(formatNumber(1284300.456, 'ms')).toBe(`1${NBSP}284${NBSP}300ms`)
    })

    test('passes undefined through', () => {
        expect(formatNumber(undefined)).toBeUndefined()
    })
})

describe('formatNumber1', () => {
    test('keeps exactly one fraction digit', () => {
        expect(formatNumber1(1284300.456)).toBe(`1${NBSP}284${NBSP}300.5`)
        expect(formatNumber1(12, '%')).toBe('12.0%')
    })

    test('passes undefined through', () => {
        expect(formatNumber1(undefined)).toBeUndefined()
    })
})

describe('formatNumber2', () => {
    test('keeps exactly two fraction digits', () => {
        expect(formatNumber2(1284300.456)).toBe(`1${NBSP}284${NBSP}300.46`)
        expect(formatNumber2(12.5, '%')).toBe('12.50%')
    })

    test('passes undefined through', () => {
        expect(formatNumber2(undefined)).toBeUndefined()
    })
})

describe('formatNumberCompact', () => {
    test('shortens to a magnitude with at most two fraction digits', () => {
        expect(formatNumberCompact(999)).toBe('999')
        expect(formatNumberCompact(1500)).toBe('1.5K')
        expect(formatNumberCompact(3044300)).toBe('3.04M')
    })

    test('groups with non-breaking spaces past the largest magnitude', () => {
        expect(formatNumberCompact(1e18)).toBe(`1${NBSP}000${NBSP}000T`)
    })

    test('passes undefined through', () => {
        expect(formatNumberCompact(undefined)).toBeUndefined()
    })
})
