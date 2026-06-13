import type { ComponentProps, FC, ReactNode, Ref } from 'react'

import { cva } from 'class-variance-authority'
import { ArrowDownRight, ArrowUpRight, Info, Minus } from 'lucide-react'

import { BarChart } from '@/shared/ui/charts/bar-chart'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/shared/ui/tooltip'
import { cn } from '@/shared/utils'

import { Card } from './card'

type MetricVariant = NonNullable<ComponentProps<typeof BarChart>['series']>[number]['variant']

type AsideTrend = 'positive' | 'negative'

const valueVariants = cva('flex items-center gap-2 font-mono text-3xl font-semibold tracking-tight', {
    variants: {
        variant: {
            primary: 'text-ink',
            success: 'text-ink',
            danger: 'text-danger',
            amber: 'text-ink',
        } satisfies Record<MetricVariant, string>,
    },
    defaultVariants: {
        variant: 'primary',
    },
})

const renderTrendAside = (values: MetricCardProps['values'], trend: AsideTrend | null): ReactNode => {
    if (trend === null || !values || values.length < 2) {
        return null
    }

    const previous = values[values.length - 2].value
    const latest = values[values.length - 1].value

    if (latest === previous) {
        return <Minus className="size-4 text-ink-tertiary" />
    }

    const increasing = latest > previous
    const isGood = trend === 'positive' ? increasing : !increasing
    const color = isGood ? 'text-success' : 'text-danger'
    const Icon = increasing ? ArrowUpRight : ArrowDownRight

    return <Icon className={cn('size-4', color)} />
}

type MetricCardProps = {
    title: string
    tooltip?: string
    aside?: ReactNode
    asideTrend?: AsideTrend | null
    value?: ReactNode
    values?: Array<{ date: string; value: number }>
    chartSize?: 'sm' | 'md' | 'lg'
    variant?: MetricVariant
    footer?: ReactNode
    className?: string
    ref?: Ref<HTMLDivElement>
}

export const MetricCard: FC<MetricCardProps> = ({
    title,
    tooltip,
    aside,
    asideTrend = 'positive',
    value,
    values,
    chartSize = 'sm',
    variant = 'primary',
    footer,
    className,
    ...props
}) => (
    <Card separated={false} className={cn('gap-3 p-5 overflow-visible', className)} {...props}>
        <div className="flex items-center justify-between">
            <div className="flex items-center gap-1.5">
                <span className="text-sm font-medium text-ink-subtle">{title}</span>
                {tooltip && (
                    <Tooltip>
                        <TooltipTrigger asChild>
                            <span className="inline-flex">
                                <Info className="size-3.5 text-ink-tertiary" />
                            </span>
                        </TooltipTrigger>
                        <TooltipContent className="max-w-56 text-pretty">{tooltip}</TooltipContent>
                    </Tooltip>
                )}
            </div>
            {aside === undefined ? renderTrendAside(values, asideTrend) : aside}
        </div>

        {value !== null && value !== undefined && value !== '' && (
            <div className={valueVariants({ variant })}>{value}</div>
        )}

        {values && (
            <BarChart
                className={chartSize === 'lg' ? 'h-32' : chartSize === 'md' ? 'h-20' : 'h-8'}
                data={values}
                series={[{ dataKey: 'value', label: title, variant }]}
                xAxisKey="date"
                tooltip
            />
        )}

        {footer != null && <span className="text-xs font-medium text-ink-subtle">{footer}</span>}
    </Card>
)
