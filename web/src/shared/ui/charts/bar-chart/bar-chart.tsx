import type { FC, HTMLAttributes, Ref } from 'react'

import { Bar, BarChart as RechartsBarChart, XAxis } from 'recharts'

import {
    type ChartConfig,
    ChartContainer,
    ChartLegend,
    ChartLegendContent,
    ChartTooltip,
    ChartTooltipContent,
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
            <RechartsBarChart data={data} margin={{ top: 0, right: 0, bottom: 0, left: 0 }} barCategoryGap="10%">
                {xAxisKey && <XAxis dataKey={xAxisKey} hide />}
                <ChartTooltip
                    cursor={{ radius: 2 }}
                    position={{ x: 0, y: 0 }}
                    content={tooltip ? <ChartTooltipContent /> : () => null}
                />
                {legend && <ChartLegend content={<ChartLegendContent />} />}
                {series.map(({ dataKey }, index) => {
                    const rounded = !stacked || index === series.length - 1

                    return (
                        <Bar
                            key={dataKey}
                            dataKey={dataKey}
                            fill={`var(--color-${dataKey})`}
                            stackId={stacked ? 'stack' : undefined}
                            radius={[...(rounded ? TOP_RADIUS : FLAT_RADIUS)]}
                        />
                    )
                })}
            </RechartsBarChart>
        </ChartContainer>
    )
}
