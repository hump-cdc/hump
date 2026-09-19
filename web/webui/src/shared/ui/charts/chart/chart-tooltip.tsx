import {
    type ComponentProps,
    type CSSProperties,
    type FC,
    type ReactNode,
    useLayoutEffect,
    useMemo,
    useRef,
    useState,
} from 'react'

import type { DefaultTooltipContentProps, TooltipValueType } from 'recharts'
import { Tooltip } from 'recharts'

import { cn } from '@/shared/utils'

import { useChart } from './chart-context'
import { getPayloadConfigFromPayload } from './get-payload-config'

type TooltipNameType = number | string

/** Gap kept between the tooltip and the chart edges when it has to be nudged inwards. */
const EDGE_PADDING = 4

export const ChartTooltip = Tooltip

/** Renders a data point's value, e.g. `(1.2) => '1.2s'`. Unlike recharts' `formatter`, it only replaces the value. */
export type ChartValueFormatter = (value: number, name: string) => ReactNode

type ChartTooltipContentProps = ComponentProps<typeof Tooltip> &
    ComponentProps<'div'> & {
        hideLabel?: boolean
        hideIndicator?: boolean
        indicator?: 'line' | 'dot' | 'dashed'
        nameKey?: string
        labelKey?: string
        valueFormatter?: ChartValueFormatter
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
    valueFormatter,
    coordinate,
}) => {
    const { config } = useChart()
    const contentRef = useRef<HTMLDivElement>(null)
    const [shift, setShift] = useState(0)

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

    const x = coordinate?.x ?? 0
    const y = coordinate?.y ?? 0

    useLayoutEffect(() => {
        const node = contentRef.current
        const container = node?.closest('.recharts-wrapper')

        if (!node || !container) {
            setShift(0)

            return
        }

        const width = node.offsetWidth
        const containerWidth = container.clientWidth

        if (!width || !containerWidth) {
            setShift(0)

            return
        }

        const left = x - width / 2
        const minLeft = EDGE_PADDING
        const maxLeft = Math.max(minLeft, containerWidth - width - EDGE_PADDING)

        setShift(Math.min(Math.max(left, minLeft), maxLeft) - left)
    }, [x])

    if (!active || !payload?.length) {
        return null
    }

    const nestLabel = payload.length === 1 && indicator !== 'dot'

    return (
        <div
            ref={contentRef}
            className="pointer-events-none"
            style={{ transform: `translate(calc(${x}px - 50% + ${shift}px), calc(${y}px - 100% - 8px))` }}
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
                            const formattedValue =
                                typeof item.value !== 'number'
                                    ? String(item.value)
                                    : valueFormatter
                                      ? valueFormatter(item.value, key)
                                      : item.value.toLocaleString()

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
                                                    'flex min-w-0 flex-1 justify-between gap-4 leading-none',
                                                    nestLabel ? 'items-end' : 'items-center',
                                                )}
                                            >
                                                <div className="grid min-w-0 gap-1.5">
                                                    {nestLabel ? tooltipLabel : null}
                                                    <span className="max-w-24 truncate text-ink-subtle leading-4">
                                                        {itemConfig?.label ?? item.name}
                                                    </span>
                                                </div>
                                                {item.value != null && (
                                                    <span className="shrink-0 font-medium font-mono text-ink tabular-nums">
                                                        {formattedValue}
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
