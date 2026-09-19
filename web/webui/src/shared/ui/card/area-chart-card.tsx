import type { ComponentProps, FC, ReactNode, Ref } from 'react'

import { cva } from 'class-variance-authority'
import { Info } from 'lucide-react'

import { AreaChart } from '@/shared/ui/charts/area-chart'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/shared/ui/tooltip'
import { cn } from '@/shared/utils'

import { Card } from './card'
import { CardContent } from './card-content'
import { CardHeader } from './card-header'

type AreaSeries = NonNullable<ComponentProps<typeof AreaChart>['series']>[number]

const swatchVariants = cva('h-1 w-5 rounded-sm', {
    variants: {
        variant: {
            primary: 'bg-primary',
            success: 'bg-success',
            danger: 'bg-danger',
            amber: 'bg-amber',
        } satisfies Record<AreaSeries['variant'], string>,
    },
    defaultVariants: {
        variant: 'primary',
    },
})

type AreaChartCardProps = {
    title: string
    subtitle?: string
    hint?: string
    legend?: boolean
    value?: ReactNode
    data: ComponentProps<typeof AreaChart>['data']
    series: AreaSeries[]
    xAxisKey?: string
    stacked?: boolean
    tooltip?: boolean
    tooltipValueFormatter?: ComponentProps<typeof AreaChart>['tooltipValueFormatter']
    chartClassName?: string
    className?: string
    ref?: Ref<HTMLDivElement>
}

export const AreaChartCard: FC<AreaChartCardProps> = ({
    title,
    subtitle,
    hint,
    legend = true,
    value,
    data,
    series,
    xAxisKey,
    stacked = false,
    tooltip = true,
    tooltipValueFormatter,
    chartClassName = 'h-56',
    className,
    ...props
}) => (
    <Card className={cn('overflow-visible', className)} {...props}>
        <CardHeader
            aside={
                legend ? (
                    <div className="flex items-center gap-3.5">
                        {series.map(({ dataKey, label, variant }) => (
                            <span
                                key={dataKey}
                                className="flex items-center gap-1.5 font-medium text-ink-subtle text-xs"
                            >
                                <span className={swatchVariants({ variant })} />
                                {label ?? dataKey}
                            </span>
                        ))}
                    </div>
                ) : undefined
            }
        >
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

        <CardContent className="flex flex-col gap-3 p-4">
            {value !== null && value !== undefined && value !== '' && (
                <div className="flex items-center gap-2 font-mono font-semibold text-3xl text-ink tracking-tight">
                    {value}
                </div>
            )}

            <AreaChart
                className={chartClassName}
                data={data}
                series={series}
                xAxisKey={xAxisKey}
                stacked={stacked}
                tooltip={tooltip}
                tooltipValueFormatter={tooltipValueFormatter}
            />
        </CardContent>
    </Card>
)
