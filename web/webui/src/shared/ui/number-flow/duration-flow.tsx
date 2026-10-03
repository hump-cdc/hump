import type { FC } from 'react'

import Flow, { NumberFlowGroup } from '@number-flow/react'

import { cn } from '@/shared/utils'

/** Caps the tens digit of minutes and seconds at 5, so 59 → 00 spins forward instead of back through 6–9. */
const SEXAGESIMAL_DIGITS = { 1: { max: 5 } }

type DurationFlowProps = {
    seconds: number
    suffix?: string
    className?: string
}

/** Animated duration: `Xh Ym Zs`, dropping leading zero units (`Ym Zs`, `Zs`). Units animate as one group. */
export const DurationFlow: FC<DurationFlowProps> = ({ seconds, suffix, className }) => {
    const total = Math.max(0, Math.round(seconds))
    const hours = Math.floor(total / 3600)
    const minutes = Math.floor((total % 3600) / 60)

    return (
        <span className={cn('font-mono', className)}>
            <NumberFlowGroup>
                {hours > 0 && (
                    <>
                        <Flow value={hours} suffix="h" />{' '}
                    </>
                )}
                {total >= 60 && (
                    <>
                        <Flow value={minutes} suffix="m" digits={SEXAGESIMAL_DIGITS} />{' '}
                    </>
                )}
                <Flow value={total % 60} suffix={`s${suffix ?? ''}`} digits={SEXAGESIMAL_DIGITS} />
            </NumberFlowGroup>
        </span>
    )
}
