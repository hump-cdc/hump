/** Abortable delay: resolves on timeout or immediately on abort. */
export const sleep = (ms: number, signal?: AbortSignal): Promise<void> =>
    new Promise<void>((resolve) => {
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
