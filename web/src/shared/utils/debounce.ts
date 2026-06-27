import { useEffect, useRef } from 'react'

export type Debounced<A extends unknown[]> = ((...args: A) => void) & { cancel: () => void }

/** Wrap `fn` so it runs only after `ms` of quiet; each call resets the timer. */
export const debounce = <A extends unknown[]>(fn: (...args: A) => void, ms: number): Debounced<A> => {
    let timer: ReturnType<typeof setTimeout> | undefined
    const debounced = (...args: A) => {
        clearTimeout(timer)
        timer = setTimeout(() => fn(...args), ms)
    }
    debounced.cancel = () => clearTimeout(timer)
    return debounced
}

/**
 * Stable {@link Debounced} wrapper of `fn`, cancelled automatically on unmount.
 *
 * The returned function keeps the same identity across renders while always
 * invoking the latest `fn`, so it is safe to call from effects and event
 * handlers without re-creating the timer.
 */
export const useDebounce = <A extends unknown[]>(fn: (...args: A) => void, ms: number): Debounced<A> => {
    const fnRef = useRef(fn)
    fnRef.current = fn

    const debouncedRef = useRef<Debounced<A>>(undefined)
    if (!debouncedRef.current) {
        debouncedRef.current = debounce((...args: A) => fnRef.current(...args), ms)
    }

    useEffect(() => () => debouncedRef.current?.cancel(), [])

    return debouncedRef.current
}
