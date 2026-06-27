/// <reference lib="webworker" />

import { client } from '@/shared/api/generated/client.gen'
import { db } from '@/shared/storage'

import type { PageToWorkerMessage, WorkerToPageMessage } from './messages'

/**
 * Run `work` in one tab at a time behind a Web Lock; a queued tab takes over when the
 * holder goes away. `work` must return once `signal` aborts, or it strands the lock.
 */
const withLeadership = async (
    name: string,
    signal: AbortSignal,
    work: (signal: AbortSignal) => Promise<void>,
): Promise<void> => {
    const locks: LockManager | undefined = navigator.locks
    if (!locks) {
        await work(signal)
        return
    }

    try {
        await locks.request(name, { signal }, () => work(signal))
    } catch (error) {
        // Aborting while queued rejects the request — teardown, not a failure.
        if (!signal.aborted) throw error
    }
}

/** What a run loop is handed once its worker holds the lock. */
export type WorkerContext = {
    /** Aborted by the page's `stop`; the loop must return once it fires. */
    signal: AbortSignal
    /** Post a typed notification to the page. */
    notify: (message: WorkerToPageMessage) => void
}

/** Everything a stream worker has to supply beyond its own loop. */
export type WorkerOptions = {
    /** Web Lock the loop is held behind, so only one tab writes. */
    lock: string
    /** Prefix for console errors — `metrics` reads as `[metrics:start]`. */
    label: string
    /** The loop itself. Must return once `context.signal` aborts. */
    run: (context: WorkerContext) => Promise<void>
    /**
     * Teardown on `stop`, for whatever the loop buffered outside the database. Runs once the
     * loop has unwound, and a restart waits for it.
     */
    onStop?: () => void | Promise<void>
}

/**
 * Wire a worker module to the page's start/stop protocol and run its loop as the elected
 * leader. Call once at the top level of a worker module.
 */
export const runWorker = ({ lock, label, run, onStop }: WorkerOptions): void => {
    // Aborted by an explicit `stop`; a tab close terminates the worker outright.
    let controller: AbortController | null = null

    // Settles once the previous loop unwound; a restart chains onto it, so a stop followed
    // by a start cannot leave two loops running.
    let unwound: Promise<void> = Promise.resolve()

    const notify = (message: WorkerToPageMessage): void => {
        self.postMessage(message)
    }

    const start = (baseUrl: string): void => {
        if (controller) return

        client.setConfig({ baseUrl })
        controller = new AbortController()
        const { signal } = controller

        // A second tab only costs one idle worker waiting behind the lock.
        unwound = unwound
            .then(() => db.open())
            .then(() => withLeadership(lock, signal, () => run({ signal, notify })))
            .catch((error: unknown) => console.error(`[${label}:start]`, error))
    }

    const stop = (): void => {
        controller?.abort()
        controller = null
        if (!onStop) return

        // Chained onto the loop so teardown races neither the final events nor a restart
        // refilling the buffers it drains.
        unwound = unwound.then(onStop).catch((error: unknown) => console.error(`[${label}:stop]`, error))
    }

    self.onmessage = (event: MessageEvent<PageToWorkerMessage>): void => {
        const message = event.data
        switch (message.type) {
            case 'start':
                start(message.baseUrl)
                break
            case 'stop':
                stop()
                break
        }
    }
}
