import { type ComponentProps, type CSSProperties, type FC, useMemo } from 'react'

import type { DefaultTooltipContentProps, TooltipValueType } from 'recharts'
import { Tooltip } from 'recharts'

import { cn } from '@/shared/utils'

import { useChart } from './chart-context'
import { getPayloadConfigFromPayload } from './get-payload-config'

type TooltipNameType = number | string

export const ChartTooltip = Tooltip

type ChartTooltipContentProps = ComponentProps<typeof Tooltip> &
    ComponentProps<'div'> & {
        hideLabel?: boolean
        hideIndicator?: boolean
        indicator?: 'line' | 'dot' | 'dashed'
        nameKey?: string
        labelKey?: string
        coordinate?: { x?: number; y?: number }
    } & Omit<DefaultTooltipContentProps<TooltipValueType, TooltipNameType>, 'accessibilityLayer'>

export const ChartTooltipContent: FC<ChartTooltipContentProps> = ({
    active,
    payload,
    className,
    indicator = 'dot',
    hideLabel = false,
    hideIndicator = false,
    label,
    labelFormatter,
    labelClassName,
    formatter,
    color,
    nameKey,
    labelKey,
    coordinate,
}) => {
    const { config } = useChart()

    const tooltipLabel = useMemo(() => {
        if (hideLabel || !payload?.length) {
            return null
        }

        const [item] = payload
        const key = `${labelKey ?? item?.dataKey ?? item?.name ?? 'value'}`
        const itemConfig = getPayloadConfigFromPayload(config, item, key)
        const value = !labelKey && typeof label === 'string' ? (config[label]?.label ?? label) : itemConfig?.label

        if (labelFormatter) {
            return <div className={cn('font-medium', labelClassName)}>{labelFormatter(value, payload)}</div>
        }

        if (!value) {
            return null
        }

        return <div className={cn('font-medium', labelClassName)}>{value}</div>
    }, [label, labelFormatter, payload, hideLabel, labelClassName, config, labelKey])

    if (!active || !payload?.length) {
        return null
    }

    const nestLabel = payload.length === 1 && indicator !== 'dot'

    const x = coordinate?.x ?? 0
    const y = coordinate?.y ?? 0

    return (
        <div
            className="pointer-events-none"
            style={{ transform: `translate(calc(${x}px - 50%), calc(${y}px - 100% - 8px))` }}
        >
            <div
                className={cn(
                    'grid min-w-32 items-start gap-1.5 rounded-lg border border-hairline bg-surface-1 px-2.5 py-1.5 text-xs shadow-xl',
                    className,
                )}
            >
                {!nestLabel ? tooltipLabel : null}
                <div className="grid gap-1.5">
                    {payload
                        .filter((item) => item.type !== 'none')
                        .map((item, index) => {
                            const key = `${nameKey ?? item.name ?? item.dataKey ?? 'value'}`
                            const itemConfig = getPayloadConfigFromPayload(config, item, key)
                            const indicatorColor = color ?? item.payload?.fill ?? item.color

                            return (
                                <div
                                    key={key}
                                    className={cn(
                                        'flex w-full flex-wrap items-stretch gap-2 [&>svg]:h-2.5 [&>svg]:w-2.5 [&>svg]:text-ink-subtle',
                                        indicator === 'dot' && 'items-center',
                                    )}
                                >
                                    {formatter && item?.value !== undefined && item.name ? (
                                        formatter(item.value, item.name, item, index, item.payload)
                                    ) : (
                                        <>
                                            {itemConfig?.icon ? (
                                                <itemConfig.icon />
                                            ) : (
                                                !hideIndicator && (
                                                    <div
                                                        className={cn(
                                                            'shrink-0 rounded-xs border-(--color-border) bg-(--color-bg)',
                                                            {
                                                                'h-2.5 w-2.5': indicator === 'dot',
                                                                'w-1': indicator === 'line',
                                                                'w-0 border-[1.5px] border-dashed bg-transparent':
                                                                    indicator === 'dashed',
                                                                'my-0.5': nestLabel && indicator === 'dashed',
                                                            },
                                                        )}
                                                        style={
                                                            {
                                                                '--color-bg': indicatorColor,
                                                                '--color-border': indicatorColor,
                                                            } as CSSProperties
                                                        }
                                                    />
                                                )
                                            )}
                                            <div
                                                className={cn(
                                                    'flex flex-1 justify-between leading-none',
                                                    nestLabel ? 'items-end' : 'items-center',
                                                )}
                                            >
                                                <div className="grid gap-1.5">
                                                    {nestLabel ? tooltipLabel : null}
                                                    <span className="text-ink-subtle">
                                                        {itemConfig?.label ?? item.name}
                                                    </span>
                                                </div>
                                                {item.value != null && (
                                                    <span className="font-mono font-medium text-ink tabular-nums">
                                                        {typeof item.value === 'number'
                                                            ? item.value.toLocaleString()
                                                            : String(item.value)}
                                                    </span>
                                                )}
                                            </div>
                                        </>
                                    )}
                                </div>
                            )
                        })}
                </div>
            </div>
        </div>
    )
}
