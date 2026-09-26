import { useState } from 'react'

type SwitchAnimation<T> = {
    /** The value switched to. */
    current: T
    /** The value switched away from while its exit animation runs, else `null`. */
    leaving: T | null
    /** Values to render: the leaving one (if any) followed by the current one. */
    views: T[]
    /** Call when the leaving view's exit animation ends to unmount it. */
    onLeaveEnd: () => void
}

/**
 * Keep the previous `value` mounted through its exit animation when `value` changes, so the
 * outgoing and incoming views can cross-fade. A further change mid-animation replaces the leaving one.
 */
export const useSwitchAnimation = <T>(value: T): SwitchAnimation<T> => {
    const [current, setCurrent] = useState(value)
    const [leaving, setLeaving] = useState<T | null>(null)
    if (value !== current) {
        setLeaving(current)
        setCurrent(value)
    }

    return {
        current,
        leaving,
        views: leaving === null ? [current] : [leaving, current],
        onLeaveEnd: () => setLeaving(null),
    }
}
