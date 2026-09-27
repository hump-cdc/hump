import { type FC, type HTMLAttributes, type ReactNode, type Ref, useState } from 'react'

import { NumberFlow } from '@/shared/ui/number-flow'
import { cn } from '@/shared/utils'

type ProgressChartProps = Omit<HTMLAttributes<HTMLDivElement>, 'title'> & {
    title: ReactNode
    /** Share of the bar filled green, 0 to 100. An absent value hides the percentage and fades the fill out, leaving the grey track. */
    value?: number
    /** Paints the unfilled remainder red rather than leaving it grey. */
    showError?: boolean
    ref?: Ref<HTMLDivElement>
}

/** Titled two-tone bar with its percentage: `value` percent in green, the remainder grey or, with `showError`, red; changes animate. */
export const ProgressChart: FC<ProgressChartProps> = ({ title, value, showError = false, className, ...props }) => {
    // The fill keeps its last width while fading out, so it does not collapse to empty mid-transition.
    const [lastValue, setLastValue] = useState(value)
    if (value !== undefined && value !== lastValue) setLastValue(value)

    const hasValue = value !== undefined
    const fill = Math.max(0, Math.min(100, value ?? lastValue ?? 0))

    return (
        <div className={cn('flex flex-col gap-2', className)} {...props}>
            {/* Fixed height keeps the chart from shifting when the percentage appears or goes away. */}
            <div className="flex h-4 items-center justify-between">
                <span className="font-medium text-ink-subtle text-xs">{title}</span>
                {hasValue && (
                    <aside className="font-medium text-success text-xs">
                        <NumberFlow value={value} kind="percent" />
                    </aside>
                )}
            </div>

            <div className="flex h-2 w-full overflow-hidden rounded-full bg-surface-3">
                <div
                    className={cn(
                        'h-full bg-success transition-[width,opacity] duration-500 ease-out',
                        !hasValue && 'opacity-0',
                    )}
                    style={{ width: `${fill}%` }}
                />
                <div
                    className={cn(
                        'h-full flex-1 bg-danger transition-opacity duration-500',
                        (!showError || !hasValue) && 'opacity-0',
                    )}
                />
            </div>
        </div>
    )
}
