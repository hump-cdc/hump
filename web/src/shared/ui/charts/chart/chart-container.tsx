import { type ComponentProps, type FC, useId } from 'react'

import { ResponsiveContainer } from 'recharts'

import { cn } from '@/shared/utils'

import { type ChartConfig, ChartContext } from './chart-context'
import { ChartStyle } from './chart-style'

const INITIAL_DIMENSION = { width: 320, height: 200 } as const

type ChartContainerProps = ComponentProps<'div'> & {
    config: ChartConfig
    children: ComponentProps<typeof ResponsiveContainer>['children']
    initialDimension?: {
        width: number
        height: number
    }
}

export const ChartContainer: FC<ChartContainerProps> = ({
    id,
    className,
    children,
    config,
    initialDimension = INITIAL_DIMENSION,
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
                    "flex aspect-video justify-center text-xs [&_.recharts-cartesian-axis-tick_text]:fill-ink-subtle [&_.recharts-cartesian-grid_line[stroke='#ccc']]:stroke-hairline [&_.recharts-curve.recharts-tooltip-cursor]:stroke-hairline [&_.recharts-dot[stroke='#fff']]:stroke-transparent [&_.recharts-layer]:outline-hidden [&_.recharts-polar-grid_[stroke='#ccc']]:stroke-hairline [&_.recharts-radial-bar-background-sector]:fill-surface-2 [&_.recharts-rectangle.recharts-tooltip-cursor]:fill-surface-2 [&_.recharts-reference-line_[stroke='#ccc']]:stroke-hairline [&_.recharts-sector]:outline-hidden [&_.recharts-sector[stroke='#fff']]:stroke-transparent [&_.recharts-surface]:outline-hidden",
                    className,
                )}
                {...props}
            >
                <ChartStyle id={chartId} config={config} />
                <ResponsiveContainer initialDimension={initialDimension}>{children}</ResponsiveContainer>
            </div>
        </ChartContext.Provider>
    )
}
