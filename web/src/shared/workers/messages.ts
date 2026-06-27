// Message protocol between the page and a stream worker, shared by both workers.

/** Messages the page sends to a worker. */
export type PageToWorkerMessage = { type: 'start'; baseUrl: string } | { type: 'stop' }

/**
 * Messages a worker sends to the page. Unused today (the page renders from IndexedDB via
 * liveQuery), reserved for nudges the store cannot deliver.
 */
export type WorkerToPageMessage = {
    type: 'notify'
    /** Why the worker is notifying, for logging. */
    reason: string
    /** TanStack Query keys the page may invalidate. */
    invalidate?: string[]
}
