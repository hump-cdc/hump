import { barY, crosshair, defineChart, group, stack } from '@tanstack/charts'
import { decorative } from '@tanstack/charts/mark/decorative'
import { RendererChart } from '@tanstack/charts/react/tooltip'
import { scaleBand } from '@tanstack/charts/scales/band'
import { scaleLinear } from '@tanstack/charts/scales/linear'

import { type FC, type HTMLAttributes, type Ref, useMemo } from 'react'

import {
    ChartContainer,
    type ChartDataRow,
    type ChartSeries,
    ChartTooltipContent,
    type ChartTooltipValue,
    chartCursorSpring,
    chartTooltipOptions,
    createChartConfig,
    getChartMaximum,
    getSeriesColor,
    toChartRows,
    useChartRenderer,
} from '@/shared/ui/charts/chart'
import { cn } from '@/shared/utils'

const BAR_RADIUS = 2
/** Share of each x step left empty between bars. */
const BAR_GAP = 0.1
/** Height in pixels of the stand-in bar drawn where a step is empty or too small to see. */
const MIN_BAR_SIZE = 1

/** Paint of the stand-in bars when there is no series to take a color from. */
const BASELINE_COLOR = 'var(--hairline-strong)'

type BarChartProps = HTMLAttributes<HTMLDivElement> & {
    data: ChartDataRow[]
    series: ChartSeries[]
    xAxisKey?: string
    stacked?: boolean
    tooltip?: boolean
    /** How each tooltip value is rendered; the kind defaults to integer or decimal by value. */
    tooltipValue?: ChartTooltipValue
    legend?: boolean
    ref?: Ref<HTMLDivElement>
}

export const BarChart: FC<BarChartProps> = ({
    className,
    data,
    series,
    xAxisKey,
    stacked = true,
    tooltip = false,
    tooltipValue,
    legend = false,
    ...props
}) => {
    const config = useMemo(() => createChartConfig(series), [series])
    const renderer = useChartRenderer()
    const seriesOrder = useMemo(() => series.map(({ dataKey }) => dataKey), [series])

    // Without a series there is nothing to hover, so the tooltip stays off like on an empty plot.
    const hasTooltip = tooltip && series.length > 0

    const definition = useMemo(
        () => createBarDefinition({ data, series, xAxisKey, stacked, tooltip: hasTooltip }),
        [data, series, xAxisKey, stacked, hasTooltip],
    )

    return (
        <ChartContainer config={config} legend={legend} className={cn('aspect-auto h-40 w-full', className)} {...props}>
            {(chartId) => (
                <RendererChart
                    definition={definition}
                    renderer={renderer}
                    idPrefix={chartId}
                    ariaLabel={seriesOrder.map((key) => config[key]?.label ?? key).join(', ') || 'Bar chart'}
                    style={{ height: '100%' }}
                    renderTooltipBody={({ points }) => (
                        <ChartTooltipContent points={points} seriesOrder={seriesOrder} valueProps={tooltipValue} />
                    )}
                />
            )}
        </ChartContainer>
    )
}

/**
 * Stacked (or grouped) bars per x step with only the outer end rounded, a band cursor behind
 * the hovered step, and a one-pixel stand-in bar wherever a step is empty or shorter than a pixel
 * so such steps stay visible.
 */
const createBarDefinition = ({
    data,
    series,
    xAxisKey,
    stacked,
    tooltip,
}: {
    data: ChartDataRow[]
    series: ChartSeries[]
    xAxisKey: string | undefined
    stacked: boolean
    tooltip: boolean
}) => {
    const rows = toChartRows({ data, series, xAxisKey, stacked })
    const maximum = getChartMaximum(rows)
    const indices = data.map((_, index) => index)
    const stepSizes = indices.map((index) => {
        const values = rows.filter((row) => row.index === index).map((row) => row.value)
        return stacked ? values.reduce((sum, value) => sum + value, 0) : Math.max(0, ...values)
    })
    const baselineColor = series.length ? getSeriesColor(series[0].dataKey) : BASELINE_COLOR

    return defineChart(
        ({ height }) => {
            const minBarValue = (maximum * MIN_BAR_SIZE) / Math.max(height, 1)
            const baselineIndices = indices.filter((index) => stepSizes[index] < minBarValue)

            return {
                marks: [
                    crosshair<number, number>({
                        x: { band: { fill: 'var(--surface-2)', fillOpacity: 1, radius: BAR_RADIUS } },
                        y: false,
                        motion: { transition: chartCursorSpring },
                    }),
                    decorative(
                        barY(baselineIndices, {
                            id: 'baseline',
                            x: (index) => index,
                            y1: 0,
                            y2: minBarValue,
                            key: (index) => String(index),
                            fill: baselineColor,
                        }),
                    ),
                    barY(rows, {
                        id: 'bars',
                        x: 'index',
                        y: 'value',
                        z: 'series',
                        key: 'key',
                        fill: (row) => getSeriesColor(row.series),
                        layout: stacked ? stack({ order: series.map(({ dataKey }) => dataKey) }) : group(),
                        radius: { end: BAR_RADIUS },
                    }),
                ],
                scales: {
                    x: {
                        scale: scaleBand<number>()
                            .domain(indices)
                            .paddingInner(BAR_GAP)
                            .paddingOuter(BAR_GAP / 2),
                        axis: false,
                    },
                    y: { scale: scaleLinear().domain([0, maximum]), axis: false },
                },
                margin: 0,
            }
        },
        {
            focus: 'group-x',
            focusRing: false,
            maxFocusDistance: Number.POSITIVE_INFINITY,
            keyboard: true,
            tooltip: tooltip ? chartTooltipOptions : false,
        },
    )
}
