import type { ChartPoint, ChartTooltipOptions } from '@tanstack/charts'
import { tooltip } from '@tanstack/charts/tooltip'
import { portal } from '@tanstack/charts/tooltip/portal'

import type { ComponentProps, FC } from 'react'

import { NumberFlow } from '@/shared/ui/number-flow'

import { useChart } from './chart-context'
import type { ChartRow } from './chart-series'

/** Gap between the tooltip and the value it floats above. */
const TOOLTIP_OFFSET = 8

/** NumberFlow props applied to every value in the tooltip, e.g. `{ kind: 'decimal', suffix: 'ms' }`. */
export type ChartTooltipValue = Pick<ComponentProps<typeof NumberFlow>, 'kind' | 'suffix'>

/**
 * Tooltip behavior shared by every chart: the popover floats centered above the highest focused
 * value, flips to another side near the edges, and is portaled so small cards cannot clip it.
 */
export const chartTooltipOptions = {
    use: tooltip,
    portal,
    anchor: (points) => {
        if (!points.length) {
            return undefined
        }

        return { x: points[0].x, y: Math.min(...points.map(({ y }) => y)) }
    },
    placement: ['top', 'right', 'left', 'bottom'],
    offset: TOOLTIP_OFFSET,
    sticky: false,
    visibility: 'focus',
} satisfies ChartTooltipOptions<ChartRow, number, number> & { use: typeof tooltip }

type ChartTooltipContentProps = {
    points: readonly ChartPoint<ChartRow, number, number>[]
    /** Ordered series keys; rows follow this order regardless of stacking. */
    seriesOrder: string[]
    valueProps?: ChartTooltipValue
}

/** Card with the focused x label and one row per series: color dot, label and formatted value. */
export const ChartTooltipContent: FC<ChartTooltipContentProps> = ({ points, seriesOrder, valueProps }) => {
    const { config } = useChart()

    if (!points.length) {
        return null
    }

    const rows = [...points]
        .map(({ datum }) => datum)
        .sort((left, right) => seriesOrder.indexOf(left.series) - seriesOrder.indexOf(right.series))
    const label = rows[0].label

    return (
        <div className="grid min-w-32 items-start gap-1.5 rounded-lg border border-hairline bg-surface-1 px-2.5 py-1.5 font-normal font-sans text-ink text-xs shadow-xl">
            {label && <div className="font-medium">{label}</div>}
            <div className="grid gap-1.5">
                {rows.map(({ series, value }) => {
                    const itemConfig = config[series]

                    return (
                        <div
                            // Keyed by series so a row survives moving to another x step and NumberFlow animates the new value.
                            key={series}
                            className="flex w-full flex-wrap items-center gap-2 [&>svg]:h-2.5 [&>svg]:w-2.5 [&>svg]:text-ink-subtle"
                        >
                            {itemConfig?.icon ? (
                                <itemConfig.icon />
                            ) : (
                                <div
                                    className="h-2.5 w-2.5 shrink-0 rounded-xs"
                                    style={{ backgroundColor: `var(--color-${series})` }}
                                />
                            )}
                            <div className="flex min-w-0 flex-1 items-center justify-between gap-4 leading-none">
                                <span className="max-w-24 truncate text-ink-subtle leading-4">
                                    {itemConfig?.label ?? series}
                                </span>
                                <span className="shrink-0 font-medium font-mono text-ink tabular-nums">
                                    <NumberFlow
                                        value={value}
                                        kind={valueProps?.kind ?? (Number.isInteger(value) ? 'integer' : 'decimal')}
                                        suffix={valueProps?.suffix}
                                    />
                                </span>
                            </div>
                        </div>
                    )
                })}
            </div>
        </div>
    )
}
