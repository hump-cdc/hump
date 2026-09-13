import { type ComponentProps, type FC, useContext } from 'react'

import type { DefaultLegendContentProps } from 'recharts'
import { Legend } from 'recharts'

import { cn } from '@/shared/utils'

import { type ChartConfig, ChartContext } from './chart-context'
import { getPayloadConfigFromPayload } from './get-payload-config'

export const ChartLegend = Legend

type ChartLegendContentProps = ComponentProps<'div'> & {
    hideIcon?: boolean
    nameKey?: string
    config?: ChartConfig
} & Partial<DefaultLegendContentProps>

export const ChartLegendContent: FC<ChartLegendContentProps> = ({
    className,
    hideIcon = false,
    payload,
    verticalAlign = 'bottom',
    nameKey,
    config: configProp,
}) => {
    const context = useContext(ChartContext)
    const config = context?.config ?? configProp ?? {}

    if (!payload?.length) {
        return null
    }

    return (
        <div
            className={cn(
                'flex items-center justify-center gap-4',
                verticalAlign === 'top' ? 'pb-3' : 'pt-3',
                className,
            )}
        >
            {payload
                .filter((item) => item.type !== 'none')
                .map((item) => {
                    const key = `${nameKey ?? item.dataKey ?? 'value'}`
                    const itemConfig = getPayloadConfigFromPayload(config, item, key)

                    return (
                        <div
                            key={key}
                            className="flex items-center gap-1.5 [&>svg]:h-3 [&>svg]:w-3 [&>svg]:text-ink-subtle"
                        >
                            {itemConfig?.icon && !hideIcon ? (
                                <itemConfig.icon />
                            ) : (
                                <div
                                    className="h-2 w-2 shrink-0 rounded-xs"
                                    style={{
                                        backgroundColor: item.color,
                                    }}
                                />
                            )}
                            {itemConfig?.label}
                        </div>
                    )
                })}
        </div>
    )
}
