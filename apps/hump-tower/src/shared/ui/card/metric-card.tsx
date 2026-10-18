import type { ComponentProps, FC, ReactNode, Ref } from 'react'

import { cva } from 'class-variance-authority'
import { ArrowDownRight, ArrowUpRight, Info } from 'lucide-react'

import { BarChart } from '@/shared/ui/charts/bar-chart'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/shared/ui/tooltip'
import { cn } from '@/shared/utils'

import { Card } from './card'
import { CardContent } from './card-content'
import { CardHeader } from './card-header'

type MetricVariant = NonNullable<ComponentProps<typeof BarChart>['series']>[number]['variant']

type AsideTrend = 'positive' | 'negative'

const valueVariants = cva('flex items-center gap-2 font-mono font-semibold text-3xl tracking-tight', {
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

    if (latest === previous) return null

    const increasing = latest > previous
    const isGood = trend === 'positive' ? increasing : !increasing
    const color = isGood ? 'text-success' : 'text-danger'
    const Icon = increasing ? ArrowUpRight : ArrowDownRight

    return <Icon className={cn('size-4', color)} />
}

type MetricCardProps = {
    title: string
    hint?: string
    aside?: ReactNode
    asideTrend?: AsideTrend | null
    value?: ReactNode
    values?: Array<{ date: string; value: number }>
    tooltipValue?: ComponentProps<typeof BarChart>['tooltipValue']
    chartSize?: 'sm' | 'md' | 'lg'
    chartClassName?: string
    variant?: MetricVariant
    separated?: boolean
    footer?: ReactNode
    className?: string
    ref?: Ref<HTMLDivElement>
}

export const MetricCard: FC<MetricCardProps> = ({
    title,
    hint,
    aside,
    asideTrend = 'positive',
    value,
    values,
    tooltipValue,
    chartSize = 'sm',
    chartClassName,
    variant = 'primary',
    separated = false,
    footer,
    className,
    ...props
}) => {
    const label = (
        <div className="flex items-center gap-1.5">
            <span className="font-medium text-ink-subtle text-sm">{title}</span>
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
        </div>
    )

    const headerAside = aside === undefined ? renderTrendAside(values, asideTrend) : aside

    const body = (
        <>
            {value !== null && value !== undefined && value !== '' && (
                <div className={valueVariants({ variant })}>{value}</div>
            )}

            {values && (
                <BarChart
                    className={cn(chartSize === 'lg' ? 'h-32' : chartSize === 'md' ? 'h-20' : 'h-8', chartClassName)}
                    data={values}
                    series={[{ dataKey: 'value', label: title, variant }]}
                    xAxisKey="date"
                    tooltip
                    tooltipValue={tooltipValue}
                />
            )}

            {footer != null && <span className="font-medium text-ink-subtle text-xs">{footer}</span>}
        </>
    )

    if (separated) {
        return (
            <Card className={cn('overflow-visible', className)} {...props}>
                <CardHeader aside={headerAside}>{label}</CardHeader>
                <CardContent className="flex flex-col gap-3 p-4">{body}</CardContent>
            </Card>
        )
    }

    return (
        <Card separated={false} className={cn('gap-3 overflow-visible p-5', className)} {...props}>
            <div className="flex items-center justify-between">
                {label}
                {headerAside}
            </div>

            {body}
        </Card>
    )
}
