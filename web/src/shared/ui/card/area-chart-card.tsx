import type { ComponentProps, FC, Ref } from 'react'

import { cva } from 'class-variance-authority'

import { AreaChart } from '@/shared/ui/charts/area-chart'
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
    legend?: boolean
    data: ComponentProps<typeof AreaChart>['data']
    series: AreaSeries[]
    xAxisKey?: string
    stacked?: boolean
    tooltip?: boolean
    chartClassName?: string
    className?: string
    ref?: Ref<HTMLDivElement>
}

export const AreaChartCard: FC<AreaChartCardProps> = ({
    title,
    subtitle,
    legend = true,
    data,
    series,
    xAxisKey,
    stacked = false,
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
            <AreaChart
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
