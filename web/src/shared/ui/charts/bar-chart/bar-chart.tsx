import type { FC, HTMLAttributes, Ref } from 'react'

import { Bar, BarChart as RechartsBarChart, XAxis, YAxis } from 'recharts'

import {
    type ChartConfig,
    ChartContainer,
    ChartLegend,
    ChartLegendContent,
    ChartTooltip,
    ChartTooltipContent,
    type ChartValueFormatter,
} from '@/shared/ui/charts/chart'
import { cn } from '@/shared/utils'

type BarVariant = 'success' | 'primary' | 'danger' | 'amber'

const variantColor: Record<BarVariant, string> = {
    success: 'var(--success)',
    primary: 'var(--primary)',
    danger: 'var(--danger)',
    amber: 'var(--amber)',
}

const TOP_RADIUS = [2, 2, 0, 0] as const
const FLAT_RADIUS = [0, 0, 0, 0] as const
const MIN_BAR_SIZE = 1
const FALLBACK_DOMAIN = [0, 1] as const

/** Data key of the synthetic series that keeps a seriesless chart from rendering blank. */
const BASELINE_KEY = '__baseline__'

type BarSeries = {
    dataKey: string
    label?: string
    variant: BarVariant
}

type BarChartProps = HTMLAttributes<HTMLDivElement> & {
    data: Array<Record<string, number | string>>
    series: BarSeries[]
    xAxisKey?: string
    stacked?: boolean
    tooltip?: boolean
    /** Formats each value shown in the tooltip, e.g. `(value) => `${value}s``. */
    tooltipValueFormatter?: ChartValueFormatter
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
    tooltipValueFormatter,
    legend = false,
    ...props
}) => {
    const config: ChartConfig = Object.fromEntries(
        series.map(({ dataKey, label, variant }) => [
            dataKey,
            { label: label ?? dataKey, color: variantColor[variant] },
        ]),
    )

    // Without a series there is nothing to plot, so stand in a zero-valued one that the
    // `minPointSize` baseline below turns into the same flat hairline an all-zero chart draws.
    const baselineOnly = series.length === 0
    const rows = baselineOnly ? data.map((entry) => ({ ...entry, [BASELINE_KEY]: 0 })) : data
    const allZero = rows.every((entry) => series.every(({ dataKey }) => Number(entry[dataKey] ?? 0) === 0))

    return (
        <ChartContainer config={config} className={cn('aspect-auto h-40 w-full', className)} {...props}>
            <RechartsBarChart data={rows} margin={{ top: 0, right: 0, bottom: 0, left: 0 }} barCategoryGap="10%">
                {xAxisKey && <XAxis dataKey={xAxisKey} hide />}
                {allZero && <YAxis domain={[...FALLBACK_DOMAIN]} allowDataOverflow hide />}
                <ChartTooltip
                    cursor={{ radius: 2 }}
                    position={{ x: 0, y: 0 }}
                    content={
                        tooltip && !baselineOnly ? (
                            <ChartTooltipContent valueFormatter={tooltipValueFormatter} />
                        ) : (
                            () => null
                        )
                    }
                />
                {legend && <ChartLegend content={<ChartLegendContent />} />}
                {baselineOnly && (
                    <Bar
                        dataKey={BASELINE_KEY}
                        fill="var(--hairline-strong)"
                        legendType="none"
                        radius={[...FLAT_RADIUS]}
                        minPointSize={() => MIN_BAR_SIZE}
                    />
                )}
                {series.map(({ dataKey }, index) => {
                    const rounded = !stacked || index === series.length - 1

                    return (
                        <Bar
                            key={dataKey}
                            dataKey={dataKey}
                            fill={`var(--color-${dataKey})`}
                            stackId={stacked ? 'stack' : undefined}
                            radius={[...(rounded ? TOP_RADIUS : FLAT_RADIUS)]}
                            minPointSize={() => MIN_BAR_SIZE}
                        />
                    )
                })}
            </RechartsBarChart>
        </ChartContainer>
    )
}
