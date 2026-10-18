import { describe, expect, test } from 'bun:test'

import {
    type ChartRow,
    type ChartSeries,
    createChartConfig,
    getChartMaximum,
    getSeriesColor,
    toChartRows,
} from './chart-series'

const series: ChartSeries[] = [
    { dataKey: 'success', label: 'Delivered', variant: 'success' },
    { dataKey: 'failed', variant: 'danger' },
]

const row = (end: number): ChartRow => ({
    key: 'k',
    index: 0,
    label: undefined,
    series: 's',
    value: end,
    start: 0,
    end,
})

describe('getSeriesColor', () => {
    test('reads the per-chart color variable of the series', () => {
        expect(getSeriesColor('success')).toBe('var(--color-success)')
    })
})

describe('createChartConfig', () => {
    test('maps each series to its label and variant color, falling back to the data key', () => {
        expect(createChartConfig(series)).toEqual({
            success: { label: 'Delivered', color: 'var(--success)' },
            failed: { label: 'failed', color: 'var(--danger)' },
        })
    })
})

describe('toChartRows', () => {
    const data = [
        { time: '10:00', success: 3, failed: 1 },
        { time: '10:01', success: 5, failed: 2 },
    ]

    test('stacks each series on top of the previous one', () => {
        const rows = toChartRows({ data, series, xAxisKey: 'time', stacked: true })

        expect(rows).toEqual([
            { key: '10:00:success', index: 0, label: '10:00', series: 'success', value: 3, start: 0, end: 3 },
            { key: '10:00:failed', index: 0, label: '10:00', series: 'failed', value: 1, start: 3, end: 4 },
            { key: '10:01:success', index: 1, label: '10:01', series: 'success', value: 5, start: 0, end: 5 },
            { key: '10:01:failed', index: 1, label: '10:01', series: 'failed', value: 2, start: 5, end: 7 },
        ])
    })

    test('starts every series at zero when not stacked', () => {
        const rows = toChartRows({ data, series, xAxisKey: 'time', stacked: false })

        expect(rows.map(({ start, end }) => [start, end])).toEqual([
            [0, 3],
            [0, 1],
            [0, 5],
            [0, 2],
        ])
    })

    test('keys rows by index when labels are missing or repeat', () => {
        const repeated = [
            { time: 'x', success: 1 },
            { time: 'x', success: 2 },
        ]

        expect(toChartRows({ data: repeated, series, xAxisKey: 'time', stacked: false }).map(({ key }) => key)).toEqual(
            ['0:success', '0:failed', '1:success', '1:failed'],
        )
        expect(toChartRows({ data, series: [series[0]], xAxisKey: undefined, stacked: false })).toMatchObject([
            { key: '0:success', label: undefined },
            { key: '1:success', label: undefined },
        ])
    })

    test('reads missing and non-numeric values as zero', () => {
        const rows = toChartRows({ data: [{ time: '10:00', success: 'n/a' }], series, xAxisKey: 'time', stacked: true })

        expect(rows.map(({ value, start, end }) => [value, start, end])).toEqual([
            [0, 0, 0],
            [0, 0, 0],
        ])
    })
})

describe('getChartMaximum', () => {
    test('rounds the largest extent up to a nice tick', () => {
        expect(getChartMaximum([row(7.3)])).toBe(8)
        expect(getChartMaximum([row(42), row(87)])).toBe(90)
        expect(getChartMaximum([row(1234)])).toBe(1300)
    })

    test('falls back to 1 for an empty or all-zero chart', () => {
        expect(getChartMaximum([])).toBe(1)
        expect(getChartMaximum([row(0)])).toBe(1)
    })
})
