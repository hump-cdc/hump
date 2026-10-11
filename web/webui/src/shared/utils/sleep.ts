/** Abortable delay: resolves on timeout, or at once when `signal` aborts or already has. */
export const sleep = (ms: number, signal?: AbortSignal): Promise<void> =>
    new Promise<void>((resolve) => {
        if (signal?.aborted) {
            resolve()
            return
        }

        const timer = setTimeout(resolve, ms)
        signal?.addEventListener(
            'abort',
            () => {
                clearTimeout(timer)
                resolve()
            },
            { once: true },
        )
    })
