import type { FC, HTMLAttributes, Ref } from 'react'

import { Area, AreaChart as RechartsAreaChart, XAxis } from 'recharts'

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

type AreaVariant = 'success' | 'primary' | 'danger' | 'amber'

const variantColor: Record<AreaVariant, string> = {
    success: 'var(--success)',
    primary: 'var(--primary)',
    danger: 'var(--danger)',
    amber: 'var(--amber)',
}

type AreaSeries = {
    dataKey: string
    label?: string
    variant: AreaVariant
}

type AreaChartProps = HTMLAttributes<HTMLDivElement> & {
    data: Array<Record<string, number | string>>
    series: AreaSeries[]
    xAxisKey?: string
    stacked?: boolean
    tooltip?: boolean
    /** Formats each value shown in the tooltip, e.g. `(value) => `${value}s``. */
    tooltipValueFormatter?: ChartValueFormatter
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

    return (
        <ChartContainer config={config} className={cn('aspect-auto h-40 w-full', className)} {...props}>
            <RechartsAreaChart data={data} margin={{ top: 0, right: 0, bottom: 0, left: 0 }}>
                <defs>
                    {series.map(({ dataKey }) => (
                        <linearGradient key={dataKey} id={`fill-${dataKey}`} x1="0" y1="0" x2="0" y2="1">
                            <stop offset="0%" stopColor={`var(--color-${dataKey})`} stopOpacity={0.35} />
                            <stop offset="100%" stopColor={`var(--color-${dataKey})`} stopOpacity={0} />
                        </linearGradient>
                    ))}
                </defs>
                {xAxisKey && <XAxis dataKey={xAxisKey} hide />}
                <ChartTooltip
                    position={{ x: 0, y: 0 }}
                    content={tooltip ? <ChartTooltipContent valueFormatter={tooltipValueFormatter} /> : () => null}
                />
                {legend && <ChartLegend content={<ChartLegendContent />} />}
                {series.map(({ dataKey }) => (
                    <Area
                        key={dataKey}
                        type="linear"
                        dataKey={dataKey}
                        stroke={`var(--color-${dataKey})`}
                        strokeWidth={2}
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        fill={`url(#fill-${dataKey})`}
                        stackId={stacked ? 'stack' : undefined}
                        dot={false}
                        activeDot={{ r: 3 }}
                    />
                ))}
            </RechartsAreaChart>
        </ChartContainer>
    )
}
