/** Wait before the first retry; each further attempt doubles it. */
export const BACKOFF_BASE_MS = 1_000

/** Cap on the per-attempt wait. */
export const BACKOFF_MAX_MS = 30_000

/** An attempt that lived this long counts as a real connection, so the next failure starts over. */
export const STABLE_ATTEMPT_MS = 10_000

/** Exponential retry delays, resettable once a connection proves itself. */
export type Backoff = {
    next: () => number
    reset: () => void
}

/** Yields `base`, `base * 2`, `base * 4`… capped at `max`. */
export const createBackoff = (base: number = BACKOFF_BASE_MS, max: number = BACKOFF_MAX_MS): Backoff => {
    let attempt = 0

    return {
        next: () => {
            const delay = Math.min(base * 2 ** attempt, max)
            attempt += 1
            return delay
        },
        reset: () => {
            attempt = 0
        },
    }
}
