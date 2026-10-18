import type { ComponentProps, FC } from 'react'

import { cn } from '@/shared/utils'

import { useChart } from './chart-context'

type ChartLegendProps = ComponentProps<'div'> & {
    hideIcon?: boolean
}

/** Row of series swatches and labels, read from the chart config, shown under the plot. */
export const ChartLegend: FC<ChartLegendProps> = ({ className, hideIcon = false, ...props }) => {
    const { config } = useChart()
    const entries = Object.entries(config)

    if (!entries.length) {
        return null
    }

    return (
        <div className={cn('flex items-center justify-center gap-4 pt-3', className)} {...props}>
            {entries.map(([key, itemConfig]) => (
                <div key={key} className="flex items-center gap-1.5 [&>svg]:h-3 [&>svg]:w-3 [&>svg]:text-ink-subtle">
                    {itemConfig.icon && !hideIcon ? (
                        <itemConfig.icon />
                    ) : (
                        <div
                            className="h-2 w-2 shrink-0 rounded-xs"
                            style={{ backgroundColor: `var(--color-${key})` }}
                        />
                    )}
                    {itemConfig.label}
                </div>
            ))}
        </div>
    )
}
