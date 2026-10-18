import { areaY, crosshair, defineChart, dot, lineY } from '@tanstack/charts'
import { decorative } from '@tanstack/charts/mark/decorative'
import { RendererChart } from '@tanstack/charts/react/tooltip'
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

const ACTIVE_DOT_RADIUS = 3

type AreaChartProps = HTMLAttributes<HTMLDivElement> & {
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

export const AreaChart: FC<AreaChartProps> = ({
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

    const definition = useMemo(
        () => createAreaDefinition({ data, series, xAxisKey, stacked, tooltip }),
        [data, series, xAxisKey, stacked, tooltip],
    )

    return (
        <ChartContainer config={config} legend={legend} className={cn('aspect-auto h-40 w-full', className)} {...props}>
            {(chartId) => (
                <RendererChart
                    definition={definition}
                    renderer={renderer}
                    idPrefix={chartId}
                    ariaLabel={seriesOrder.map((key) => config[key]?.label ?? key).join(', ')}
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
 * One gradient-filled area and stroke per series over the row index, plus invisible dots that
 * carry focus and light up as the active point, and a vertical hover rule.
 */
const createAreaDefinition = ({
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
    const bySeries = series.map(({ dataKey }) => ({
        dataKey,
        rows: rows.filter((row) => row.series === dataKey),
    }))

    return defineChart({
        marks: [
            crosshair<number, number>({
                x: { stroke: 'var(--hairline)', strokeOpacity: 1, strokeWidth: 1 },
                y: false,
                motion: { transition: chartCursorSpring },
            }),
            ...bySeries.flatMap(({ dataKey, rows: seriesRows }) => [
                decorative(
                    areaY(seriesRows, {
                        id: `area-${dataKey}`,
                        x: 'index',
                        y1: 'start',
                        y2: 'end',
                        key: 'key',
                        fill: `url(#fill-${dataKey})`,
                        fillOpacity: 1,
                    }),
                ),
                decorative(
                    lineY(seriesRows, {
                        id: `line-${dataKey}`,
                        x: 'index',
                        y: 'end',
                        key: 'key',
                        stroke: getSeriesColor(dataKey),
                        strokeWidth: 2,
                    }),
                ),
            ]),
            ...bySeries.map(({ dataKey, rows: seriesRows }) =>
                dot(seriesRows, {
                    id: `dot-${dataKey}`,
                    x: 'index',
                    y: 'end',
                    // Distinct groups keep every series in the focused x group instead of deduplicating to one.
                    z: 'series',
                    key: 'key',
                    r: ACTIVE_DOT_RADIUS,
                    fill: getSeriesColor(dataKey),
                    fillOpacity: 0,
                    strokeOpacity: 0,
                    states: [{ when: { focus: 'group' }, style: { fillOpacity: 1 } }],
                }),
            ),
        ],
        scales: {
            x: { scale: scaleLinear().domain([0, Math.max(1, data.length - 1)]), axis: false },
            y: { scale: scaleLinear().domain([0, getChartMaximum(rows)]), axis: false },
        },
        gradients: series.map(({ dataKey }) => ({
            id: `fill-${dataKey}`,
            x1: 0,
            y1: 0,
            x2: 0,
            y2: 1,
            stops: [
                { offset: 0, color: getSeriesColor(dataKey), opacity: 0.35 },
                { offset: 1, color: getSeriesColor(dataKey), opacity: 0 },
            ],
        })),
        margin: 0,
        clip: true,
        focus: 'group-x',
        focusRing: false,
        maxFocusDistance: Number.POSITIVE_INFINITY,
        keyboard: true,
        tooltip: tooltip ? chartTooltipOptions : false,
    })
}
