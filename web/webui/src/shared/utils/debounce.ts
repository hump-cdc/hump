import { useEffect, useRef } from 'react'

export type Debounced<A extends unknown[]> = ((...args: A) => void) & { cancel: () => void }

/** Options of {@link debounce}. */
export type DebounceOptions = {
    /** Longest a call may be deferred, in ms, however often it is re-triggered. */
    maxWait?: number
}

/**
 * Wrap `fn` so it runs only after `ms` of quiet; each call resets the timer. With
 * `maxWait`, a steady stream of calls still runs `fn` at least that often.
 */
export const debounce = <A extends unknown[]>(
    fn: (...args: A) => void,
    ms: number,
    { maxWait }: DebounceOptions = {},
): Debounced<A> => {
    let timer: ReturnType<typeof setTimeout> | undefined
    let firstCallAt: number | undefined

    const run = (args: A) => {
        clearTimeout(timer)
        firstCallAt = undefined
        fn(...args)
    }
    const debounced = (...args: A) => {
        const now = Date.now()
        firstCallAt ??= now
        const delay = maxWait === undefined ? ms : Math.min(ms, Math.max(0, firstCallAt + maxWait - now))
        clearTimeout(timer)
        timer = setTimeout(() => run(args), delay)
    }
    debounced.cancel = () => {
        clearTimeout(timer)
        firstCallAt = undefined
    }
    return debounced
}

/**
 * Stable {@link Debounced} wrapper of `fn`, cancelled automatically on unmount.
 *
 * The returned function keeps the same identity across renders while always
 * invoking the latest `fn`, so it is safe to call from effects and event
 * handlers without re-creating the timer.
 */
export const useDebounce = <A extends unknown[]>(
    fn: (...args: A) => void,
    ms: number,
    options?: DebounceOptions,
): Debounced<A> => {
    const fnRef = useRef(fn)
    fnRef.current = fn

    const debouncedRef = useRef<Debounced<A>>(undefined)
    if (!debouncedRef.current) {
        debouncedRef.current = debounce((...args: A) => fnRef.current(...args), ms, options)
    }

    useEffect(() => () => debouncedRef.current?.cancel(), [])

    return debouncedRef.current
}
