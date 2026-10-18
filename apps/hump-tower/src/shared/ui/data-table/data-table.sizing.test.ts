import { describe, expect, test } from 'bun:test'

import { resolveColumnWidths } from './data-table.sizing'
import type { DataTableColumn } from './data-table.types'

type Row = { id: string }

describe('resolveColumnWidths', () => {
    test('flex columns share what fixed columns leave, by weight', () => {
        const columns: DataTableColumn<Row>[] = [{ width: 100 }, { flex: 1 }, { flex: 2 }]

        expect(resolveColumnWidths(columns, {}, 400)).toEqual([100, 100, 200])
    })

    test('rounds and gives the remainder to the last flex column', () => {
        const columns: DataTableColumn<Row>[] = [{ width: 666 }, { flex: 1, minWidth: 120 }, { flex: 2, minWidth: 160 }]

        expect(resolveColumnWidths(columns, {}, 1391)).toEqual([666, 242, 483])
    })

    test('pins a flex column to its bounds and re-splits the rest', () => {
        const columns: DataTableColumn<Row>[] = [{ flex: 1, maxWidth: 50 }, { flex: 1 }]

        expect(resolveColumnWidths(columns, {}, 300)).toEqual([50, 250])
    })

    test('overflows rather than squash a flex column below its minimum', () => {
        const columns: DataTableColumn<Row>[] = [{ width: 300 }, { flex: 1, minWidth: 160 }]

        expect(resolveColumnWidths(columns, {}, 350)).toEqual([300, 160])
    })

    test('applies a resize to fixed columns but never to flex ones', () => {
        const columns: DataTableColumn<Row>[] = [{ width: 100, minWidth: 80 }, { flex: 1 }]

        expect(resolveColumnWidths(columns, { col_0: 50, col_1: 10 }, 400)).toEqual([80, 320])
    })

    test('auto-fits every column by width when none sets flex, a resized one staying fixed', () => {
        const columns: DataTableColumn<Row>[] = [{ width: 100 }, { width: 300 }]

        expect(resolveColumnWidths(columns, {}, 800)).toEqual([200, 600])
        expect(resolveColumnWidths(columns, {}, 200)).toEqual([100, 300])
        expect(resolveColumnWidths(columns, { col_0: 150 }, 800)).toEqual([150, 650])
    })
})
