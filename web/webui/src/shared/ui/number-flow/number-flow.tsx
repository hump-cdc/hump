import type { FC } from 'react'

import Flow, { type Format } from '@number-flow/react'

import { cn, scaleBytes, scaleLatency } from '@/shared/utils'

import { DurationFlow } from './duration-flow'

type Kind = 'integer' | 'decimal' | 'compact' | 'percent' | 'bytes' | 'latency' | 'duration'

/** Kinds that format into a single animated number; `duration` spans several units and renders through `DurationFlow`. */
type ScalarKind = Exclude<Kind, 'duration'>

/** What NumberFlow renders: the value to animate plus how to format it. */
type Resolved = { value: number; format: Format; locales: string; suffix?: string }

/** `en-ZA` groups with a non-breaking space, matching `formatNumber`; used only where no decimal separator shows. */
const GROUPED_LOCALE = 'en-ZA'
const DEFAULT_LOCALE = 'en-US'

/** Vertical padding NumberFlow puts around digits (its digit and inner-number paddings, each half the mask height); the dash matches it so both states share one height. */
const FLOW_PADDING_BLOCK = 'calc(round(nearest, calc(var(--number-flow-mask-height, 0.25em) / 2), 1px) * 2)'

/** Fixed fraction digits without grouping; unit-scaled values stay below the grouping boundary except at rounding edges (999.6ms → `1000ms`). */
const fixed = (fractionDigits: number): Format => ({
    minimumFractionDigits: fractionDigits,
    maximumFractionDigits: fractionDigits,
    useGrouping: false,
})

/** Mirrors the string formatters in `@/shared/utils`, so an animated value reads the same as a formatted one. */
const resolve = (value: number, kind: ScalarKind): Resolved => {
    switch (kind) {
        case 'integer':
            return { value, format: { maximumFractionDigits: 0 }, locales: GROUPED_LOCALE }
        case 'decimal':
            return { value, format: { maximumFractionDigits: 2, useGrouping: false }, locales: DEFAULT_LOCALE }
        case 'compact':
            return {
                value,
                format:
                    Math.abs(value) >= 999.995
                        ? { notation: 'compact', minimumFractionDigits: 2, maximumFractionDigits: 2 }
                        : { notation: 'compact', maximumFractionDigits: 2 },
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
    /** Rendered as a dash while undefined. Seconds for `duration`. */
    value: number | undefined
    kind?: Kind
    /** Appended after the kind's own suffix. */
    suffix?: string
    className?: string
}

/** Animated number that inherits the surrounding typography; only the digit transitions are added. */
export const NumberFlow: FC<NumberFlowProps> = ({ value, kind = 'integer', suffix, className }) => {
    if (value === undefined) {
        return (
            <span
                className={cn('inline-block font-mono leading-none', className)}
                style={{ paddingBlock: FLOW_PADDING_BLOCK }}
            >
                —
            </span>
        )
    }

    if (kind === 'duration') {
        return <DurationFlow seconds={value} suffix={suffix} className={className} />
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
