import { describe, expect, test } from 'bun:test'

import { formatBytes, scaleBytes } from './format.bytes'

describe('formatBytes', () => {
    test('prints whole bytes below one kilobyte', () => {
        expect(formatBytes(0)).toBe('0 B')
        expect(formatBytes(512)).toBe('512 B')
    })

    test('scales by 1024 with two fraction digits', () => {
        expect(formatBytes(1536)).toBe('1.50 KB')
        expect(formatBytes(1500 * 1024 * 1024)).toBe('1.46 GB')
        expect(formatBytes(5 * 1024 ** 3)).toBe('5.00 GB')
    })

    test('keeps the sign of a negative size', () => {
        expect(formatBytes(-1536)).toBe('-1.50 KB')
    })

    test('stops at TB and groups thousands with a plain space', () => {
        expect(formatBytes(1024 ** 6)).toBe('1 048 576.00 TB')
    })

    test('compares against the unit boundary before rounding', () => {
        expect(formatBytes(1023.6)).toBe('1024 B')
    })

    test('passes undefined through', () => {
        expect(formatBytes(undefined)).toBeUndefined()
    })
})

describe('scaleBytes', () => {
    test('splits the scaled value from its unit', () => {
        expect(scaleBytes(1536)).toEqual({ value: 1.5, suffix: ' KB', fractionDigits: 2 })
    })

    test('rounds whole bytes and asks for no fraction digits', () => {
        expect(scaleBytes(512.4)).toEqual({ value: 512, suffix: ' B', fractionDigits: 0 })
    })

    test('keeps the sign of a negative size', () => {
        expect(scaleBytes(-2048)).toEqual({ value: -2, suffix: ' KB', fractionDigits: 2 })
    })

    test('stops at TB', () => {
        expect(scaleBytes(1024 ** 6)).toEqual({ value: 1024 ** 2, suffix: ' TB', fractionDigits: 2 })
    })
})
