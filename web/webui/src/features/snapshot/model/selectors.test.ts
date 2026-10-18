import { describe, expect, test } from 'bun:test'

import { aggregate, snapshotQuery } from './selectors'
import type { Sink, Table } from './types'

const sink = (count: number): Sink => ({ type: 'kafka', host: 'localhost:9092', target: 'orders', count })

const table = (rowsRead: number, totalRows: number, sinks: Sink[] = []): Table => ({
    id: 'public.orders',
    schema: 'public',
    table: 'orders',
    name: 'public.orders',
    where: '',
    status: 'processing',
    rowsRead,
    totalRows,
    sinks,
})

describe('snapshotQuery', () => {
    test('reads every row without a filter', () => {
        expect(snapshotQuery({ name: 'public.orders' }, '')).toBe('SELECT *\nFROM   public.orders')
        expect(snapshotQuery({ name: 'public.orders' }, '  \n ')).toBe('SELECT *\nFROM   public.orders')
    })

    test('appends the trimmed filter', () => {
        expect(snapshotQuery({ name: 'public.orders' }, '  id > 10\n')).toBe(
            'SELECT *\nFROM   public.orders\nWHERE  id > 10',
        )
    })
})

describe('aggregate', () => {
    test('sums rows and the items sent to every sink', () => {
        expect(aggregate([table(20, 100, [sink(20), sink(15)]), table(30, 50, [sink(30)])])).toEqual({
            totalRows: 150,
            rowsRead: 50,
            itemsSent: 65,
            percent: (50 / 150) * 100,
        })
    })

    test('reports 0% without an estimate', () => {
        expect(aggregate([table(10, 0)])).toEqual({ totalRows: 0, rowsRead: 10, itemsSent: 0, percent: 0 })
        expect(aggregate([])).toEqual({ totalRows: 0, rowsRead: 0, itemsSent: 0, percent: 0 })
    })

    test('caps the percent at 100 when more rows are read than estimated', () => {
        expect(aggregate([table(120, 100)]).percent).toBe(100)
    })
})
