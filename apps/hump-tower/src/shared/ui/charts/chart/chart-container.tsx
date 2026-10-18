import { type ComponentProps, type CSSProperties, type FC, type ReactNode, useId } from 'react'

import { cn } from '@/shared/utils'

import { type ChartConfig, ChartContext } from './chart-context'
import { ChartLegend } from './chart-legend'
import { ChartStyle } from './chart-style'

/**
 * Strips the native tooltip chrome so the React card rendered into it is the only visible surface.
 * Set on the container because the portaled tooltip stays its DOM descendant and inherits them.
 */
const tooltipSurfaceStyle = {
    '--ts-chart-tooltip-background': 'transparent',
    '--ts-chart-tooltip-border': 'none',
    '--ts-chart-tooltip-shadow': 'none',
    '--ts-chart-tooltip-padding': '0',
    '--ts-chart-tooltip-max-width': 'none',
    '--ts-chart-tooltip-color': 'var(--ink)',
} as CSSProperties

type ChartContainerProps = Omit<ComponentProps<'div'>, 'children'> & {
    config: ChartConfig
    legend?: boolean
    /** Renders the chart with the resource id prefix it must pass to keep gradients unique per chart. */
    children: (chartId: string) => ReactNode
}

/**
 * Sized frame of a chart: injects the per-series `--color-<key>` variables, lets the plot fill
 * the height the consumer sets through `className`, and appends the legend below it.
 */
export const ChartContainer: FC<ChartContainerProps> = ({
    id,
    className,
    style,
    config,
    legend = false,
    children,
    ...props
}) => {
    const uniqueId = useId()
    const chartId = `chart-${id ?? uniqueId.replace(/:/g, '')}`

    return (
        <ChartContext.Provider value={{ config }}>
            <div
                data-slot="chart"
                data-chart={chartId}
                className={cn(
                    // The tooltip opens as a native popover, whose UA `overflow: auto` shows a scrollbar while NumberFlow digits roll past its edges.
                    'flex aspect-video flex-col text-xs [&_.ts-chart-tooltip]:overflow-visible',
                    className,
                )}
                style={{ ...tooltipSurfaceStyle, ...style }}
                {...props}
            >
                <ChartStyle id={chartId} config={config} />
                <div className="min-h-0 flex-1 [&_.ts-chart-host]:outline-hidden [&_svg]:outline-hidden">
                    {children(chartId)}
                </div>
                {legend && <ChartLegend />}
            </div>
        </ChartContext.Provider>
    )
}
