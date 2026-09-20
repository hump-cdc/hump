import { scaleLinear } from '@tanstack/charts/scales/linear'

import type { ChartConfig } from './chart-context'

/** Semantic color a chart series is painted with. */
export type ChartVariant = 'success' | 'primary' | 'danger' | 'amber'

/** One plotted column of the wide `data` rows a chart receives. */
export type ChartSeries = {
    dataKey: string
    label?: string
    variant: ChartVariant
}

/** Wide input row: one entry per x position, one column per series. */
export type ChartDataRow = Record<string, number | string>

/** Long row the marks plot: one entry per x position and series, with its stacked extent. */
export type ChartRow = {
    key: string
    index: number
    label: string | undefined
    series: string
    value: number
    start: number
    end: number
}

const variantColor: Record<ChartVariant, string> = {
    success: 'var(--success)',
    primary: 'var(--primary)',
    danger: 'var(--danger)',
    amber: 'var(--amber)',
}

/** Paint of a series, resolved from the per-chart `--color-<dataKey>` variable `ChartStyle` injects. */
export const getSeriesColor = (dataKey: string): string => `var(--color-${dataKey})`

/** Builds the label and color config `ChartStyle`, the legend and the tooltip read. */
export const createChartConfig = (series: ChartSeries[]): ChartConfig =>
    Object.fromEntries(
        series.map(({ dataKey, label, variant }) => [
            dataKey,
            { label: label ?? dataKey, color: variantColor[variant] },
        ]),
    )

/**
 * Unfolds wide rows into one row per x position and series. `start`/`end` hold the stacked
 * extent when `stacked`, otherwise every series starts at zero.
 */
export const toChartRows = ({
    data,
    series,
    xAxisKey,
    stacked,
}: {
    data: ChartDataRow[]
    series: ChartSeries[]
    xAxisKey: string | undefined
    stacked: boolean
}): ChartRow[] => {
    const labels = data.map((entry) => (xAxisKey === undefined ? undefined : String(entry[xAxisKey] ?? '')))
    const hasUniqueLabels = labels.every((label) => label !== undefined) && new Set(labels).size === labels.length

    return data.flatMap((entry, index) => {
        let offset = 0

        return series.map(({ dataKey }) => {
            const value = Number(entry[dataKey] ?? 0) || 0
            const start = stacked ? offset : 0
            offset += value
            const identity = hasUniqueLabels ? labels[index] : String(index)

            return {
                key: `${identity}:${dataKey}`,
                index,
                label: labels[index],
                series: dataKey,
                value,
                start,
                end: start + value,
            }
        })
    })
}

/**
 * Top of the y domain: the largest stacked extent rounded up to a nice tick, falling back to 1
 * so an all-zero chart keeps a usable domain.
 */
export const getChartMaximum = (rows: ChartRow[]): number => {
    const maximum = Math.max(0, ...rows.map(({ end }) => end)) || 1

    return scaleLinear().domain([0, maximum]).nice().domain()[1] ?? maximum
}
