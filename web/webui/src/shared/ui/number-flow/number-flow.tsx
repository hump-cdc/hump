import type { FC } from 'react'

import Flow, { type Format } from '@number-flow/react'

import { cn, scaleBytes, scaleLatency } from '@/shared/utils'

type Kind = 'integer' | 'decimal' | 'compact' | 'percent' | 'bytes' | 'latency'

/** What NumberFlow renders: the value to animate plus how to format it. */
type Resolved = { value: number; format: Format; locales: string; suffix?: string }

/** `en-ZA` groups with a non-breaking space, matching `formatNumber`; used only where no decimal separator shows. */
const GROUPED_LOCALE = 'en-ZA'
const DEFAULT_LOCALE = 'en-US'

/** Fixed fraction digits without grouping; unit-scaled values stay below the grouping boundary except at rounding edges (999.6ms → `1000ms`). */
const fixed = (fractionDigits: number): Format => ({
    minimumFractionDigits: fractionDigits,
    maximumFractionDigits: fractionDigits,
    useGrouping: false,
})

/** Mirrors the string formatters in `@/shared/utils`, so an animated value reads the same as a formatted one. */
const resolve = (value: number, kind: Kind): Resolved => {
    switch (kind) {
        case 'integer':
            return { value, format: { maximumFractionDigits: 0 }, locales: GROUPED_LOCALE }
        case 'decimal':
            return { value, format: { maximumFractionDigits: 2, useGrouping: false }, locales: DEFAULT_LOCALE }
        case 'compact':
            return {
                value,
                format: { notation: 'compact', maximumFractionDigits: 2 },
                locales: DEFAULT_LOCALE,
            }
        case 'percent':
            return { value, format: { maximumFractionDigits: 0 }, locales: GROUPED_LOCALE, suffix: '%' }
        case 'bytes': {
            const scaled = scaleBytes(value)
            return {
                value: scaled.value,
                format: fixed(scaled.fractionDigits),
                locales: DEFAULT_LOCALE,
                suffix: scaled.suffix,
            }
        }
        case 'latency': {
            const scaled = scaleLatency(value)
            return {
                value: scaled.value,
                format: fixed(scaled.fractionDigits),
                locales: DEFAULT_LOCALE,
                suffix: scaled.suffix,
            }
        }
    }
}

type NumberFlowProps = {
    /** Rendered as a dash while undefined. */
    value: number | undefined
    kind?: Kind
    /** Appended after the kind's own suffix. */
    suffix?: string
    className?: string
}

/** Animated number that inherits the surrounding typography; only the digit transitions are added. */
export const NumberFlow: FC<NumberFlowProps> = ({ value, kind = 'integer', suffix, className }) => {
    if (value === undefined) {
        return <span className={cn('font-mono', className)}>—</span>
    }

    const resolved = resolve(value, kind)

    return (
        <Flow
            className={cn('font-mono', className)}
            value={resolved.value}
            format={resolved.format}
            locales={resolved.locales}
            suffix={`${resolved.suffix ?? ''}${suffix ?? ''}` || undefined}
        />
    )
}
