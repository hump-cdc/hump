import type { ComponentProps, FC, Ref } from 'react'

import { cva } from 'class-variance-authority'

import { BarChart } from '@/shared/ui/charts/bar-chart'
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

type BarChartCardProps = {
    title: string
    subtitle?: string
    legend?: boolean
    data: ComponentProps<typeof BarChart>['data']
    series: BarSeries[]
    xAxisKey?: string
    stacked?: boolean
    tooltip?: boolean
    chartClassName?: string
    className?: string
    ref?: Ref<HTMLDivElement>
}

export const BarChartCard: FC<BarChartCardProps> = ({
    title,
    subtitle,
    legend = true,
    data,
    series,
    xAxisKey,
    stacked = true,
    tooltip = true,
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
                                className="flex items-center gap-1.5 text-xs font-medium text-ink-subtle"
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
                <span className="text-sm font-semibold tracking-tight text-ink">{title}</span>
                {subtitle && <span className="font-mono text-xs text-ink-tertiary">{subtitle}</span>}
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
            />
        </CardContent>
    </Card>
)
