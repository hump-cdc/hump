import { describe, expect, test } from 'bun:test'

import { type BlockPage, blocksForRange, clampBlocks, pickTotal, rowAt, sameBlocks } from './blocks'

describe('blocksForRange', () => {
    test('covers the range and always includes block 0', () => {
        expect(blocksForRange(0, 30, 100)).toEqual([0])
        expect(blocksForRange(95, 105, 100)).toEqual([0, 1])
        expect(blocksForRange(250, 420, 100)).toEqual([0, 2, 3, 4])
    })

    test('splits exactly at the block boundary', () => {
        expect(blocksForRange(99, 99, 100)).toEqual([0])
        expect(blocksForRange(100, 100, 100)).toEqual([0, 1])
    })
})

describe('clampBlocks', () => {
    test('drops blocks past the total', () => {
        expect(clampBlocks([0, 2, 3, 4], 301, 100)).toEqual([0, 2, 3])
        expect(clampBlocks([0, 2, 3], 300, 100)).toEqual([0, 2])
    })

    test('keeps block 0 for an empty log', () => {
        expect(clampBlocks([0, 1], 0, 100)).toEqual([0])
    })
})

describe('sameBlocks', () => {
    test('compares by content', () => {
        expect(sameBlocks([0, 1], [0, 1])).toBe(true)
        expect(sameBlocks([0, 1], [0, 2])).toBe(false)
        expect(sameBlocks([0], [0, 1])).toBe(false)
    })
})

describe('pickTotal', () => {
    test('takes the total of the newest page', () => {
        expect(pickTotal([page(0, 10, 1), undefined, page(100, 12, 3), page(200, 11, 2)])).toBe(12)
    })

    test('is undefined before any page arrived', () => {
        expect(pickTotal([undefined])).toBeUndefined()
    })
})

describe('rowAt', () => {
    const pages = new Map([
        [0, page(0, 250, 1, ['a', 'b'])],
        [2, page(200, 250, 1, ['c'])],
    ])

    test('finds a row within its block', () => {
        expect(rowAt(pages, 1, 100)).toBe('b')
        expect(rowAt(pages, 200, 100)).toBe('c')
    })

    test('is undefined for an unloaded block or past a short one', () => {
        expect(rowAt(pages, 150, 100)).toBeUndefined()
        expect(rowAt(pages, 201, 100)).toBeUndefined()
    })

    test('ignores a page clamped to a different start', () => {
        expect(rowAt(new Map([[1, page(50, 50, 1, ['x'])]]), 100, 100)).toBeUndefined()
    })
})

const page = (from: number, total: number, updatedAt: number, rows: string[] = []): BlockPage<string> => ({
    from,
    total,
    updatedAt,
    rows,
})
