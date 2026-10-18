import { type FC, useEffect, useState } from 'react'

import { NumberFlow } from '@/shared/ui/number-flow'

const TICK_MS = 1_000

type ElapsedTimeProps = {
    startedAt: Date
    finishedAt?: Date
    className?: string
}

/** Animated time passed between `startedAt` and `finishedAt`, or until now while still running. */
export const ElapsedTime: FC<ElapsedTimeProps> = ({ startedAt, finishedAt, className }) => {
    const running = finishedAt === undefined
    const [now, setNow] = useState(() => Date.now())

    useEffect(() => {
        if (!running) return

        setNow(Date.now())
        const timer = setInterval(() => setNow(Date.now()), TICK_MS)
        return () => clearInterval(timer)
    }, [running])

    const end = finishedAt?.getTime() ?? now

    return <NumberFlow value={(end - startedAt.getTime()) / 1000} kind="duration" className={className} />
}
