import { describe, expect, test } from 'bun:test'

import { parseColumnSizing } from './data-table.column-state'

describe('parseColumnSizing', () => {
    test('keeps the widths of known columns', () => {
        expect(parseColumnSizing({ col_0: 120, col_9: 50, col_1: 'x' }, ['col_0', 'col_1'])).toEqual({ col_0: 120 })
    })

    test('reads the array format the AG Grid table stored, skipping flex entries', () => {
        const stored = [
            { colId: 'col_0', width: 160, flex: null },
            { colId: 'col_1', width: 300, flex: 1 },
        ]

        expect(parseColumnSizing(stored, ['col_0', 'col_1'])).toEqual({ col_0: 160 })
    })

    test('ignores anything else', () => {
        expect(parseColumnSizing('junk', ['col_0'])).toEqual({})
        expect(parseColumnSizing(null, ['col_0'])).toEqual({})
    })
})
