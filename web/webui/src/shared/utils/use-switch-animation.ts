import { createContext, useContext, useState } from 'react'

type SwitchAnimation<T> = {
    current: T
    leaving: T | null
    views: T[]
    onLeaveEnd: () => void
}

/**
 * Value a view of a switch animation renders: the live one, or, while the view fades out after `value` moved on,
 * the last value it had.
 */
export const SwitchViewValueContext = createContext<unknown>(undefined)

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

/** Value of the switch animation view enclosing the caller; `T` is not checked against what the provider holds. */
export const useSwitchViewValue = <T>(): T | undefined => useContext(SwitchViewValueContext) as T | undefined
