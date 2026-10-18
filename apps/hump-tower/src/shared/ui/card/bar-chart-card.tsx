import type { ComponentProps, FC, Ref } from 'react'

import { cva } from 'class-variance-authority'
import { Info } from 'lucide-react'

import { BarChart } from '@/shared/ui/charts/bar-chart'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/shared/ui/tooltip'
import { cn } from '@/shared/utils'

import { Card } from './card'
import { CardContent } from './card-content'
import { CardHeader } from './card-header'

type BarSeries = NonNullable<ComponentProps<typeof BarChart>['series']>[number]

const swatchVariants = cva('size-2.5 rounded-xs', {
    variants: {
        variant: {
            primary: 'bg-primary',
            success: 'bg-success',
            danger: 'bg-danger',
            amber: 'bg-amber',
        } satisfies Record<BarSeries['variant'], string>,
    },
    defaultVariants: {
        variant: 'primary',
    },
})

const INLINE_LEGEND_LIMIT = 2
const TRIGGER_SWATCH_LIMIT = 5

const LegendItem: FC<{ item: BarSeries }> = ({ item: { dataKey, label, variant } }) => (
    <span className="flex items-center gap-1.5 font-medium text-xs">
        <span className={swatchVariants({ variant })} />
        {label ?? dataKey}
    </span>
)

// Up to INLINE_LEGEND_LIMIT series fit inline; more collapse into swatches + count with the full list on hover.
const Legend: FC<{ series: BarSeries[] }> = ({ series }) => {
    if (series.length <= INLINE_LEGEND_LIMIT) {
        return (
            <div className="flex items-center gap-3.5 text-ink-subtle">
                {series.map((item) => (
                    <LegendItem key={item.dataKey} item={item} />
                ))}
            </div>
        )
    }

    return (
        <Tooltip>
            <TooltipTrigger asChild>
                <span className="flex cursor-default items-center gap-1.5 font-medium text-ink-subtle text-xs">
                    <span className="flex items-center gap-0.5">
                        {series.slice(0, TRIGGER_SWATCH_LIMIT).map(({ dataKey, variant }) => (
                            <span key={dataKey} className={swatchVariants({ variant })} />
                        ))}
                    </span>
                    {series.length} series
                </span>
            </TooltipTrigger>
            <TooltipContent side="bottom" align="end" className="flex flex-col gap-1.5 py-2">
                {series.map((item) => (
                    <LegendItem key={item.dataKey} item={item} />
                ))}
            </TooltipContent>
        </Tooltip>
    )
}

type BarChartCardProps = {
    title: string
    subtitle?: string
    hint?: string
    legend?: boolean
    data: ComponentProps<typeof BarChart>['data']
    series: BarSeries[]
    xAxisKey?: string
    stacked?: boolean
    tooltip?: boolean
    tooltipValue?: ComponentProps<typeof BarChart>['tooltipValue']
    chartClassName?: string
    className?: string
    ref?: Ref<HTMLDivElement>
}

export const BarChartCard: FC<BarChartCardProps> = ({
    title,
    subtitle,
    hint,
    legend = true,
    data,
    series,
    xAxisKey,
    stacked = true,
    tooltip = true,
    tooltipValue,
    chartClassName = 'h-56',
    className,
    ...props
}) => (
    <Card className={cn('overflow-visible', className)} {...props}>
        <CardHeader aside={legend ? <Legend series={series} /> : undefined}>
            <div className="flex items-center gap-2">
                <span className="flex items-center gap-1.5">
                    <span className="font-semibold text-ink text-sm tracking-tight">{title}</span>
                    {hint && (
                        <Tooltip>
                            <TooltipTrigger asChild>
                                <span className="inline-flex">
                                    <Info className="size-3.5 text-ink-tertiary" />
                                </span>
                            </TooltipTrigger>
                            <TooltipContent className="max-w-56 text-pretty">{hint}</TooltipContent>
                        </Tooltip>
                    )}
                </span>
                {subtitle && <span className="font-mono text-ink-tertiary text-xs">{subtitle}</span>}
            </div>
        </CardHeader>

        <CardContent className="p-4">
            <BarChart
                className={chartClassName}
                data={data}
                series={series}
                xAxisKey={xAxisKey}
                stacked={stacked}
                tooltip={tooltip}
                tooltipValue={tooltipValue}
            />
        </CardContent>
    </Card>
)
