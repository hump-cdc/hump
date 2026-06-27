import type { PageToWorkerMessage, WorkerToPageMessage } from './messages'

const NOTIFY_EVENT = 'notify'
const STATE_EVENT = 'state'

/**
 * Main-thread owner of a stream worker: starts/stops it and re-emits its notifications as
 * DOM events. The worker lives as long as the document, so a tab close tears it down.
 */
export class WorkerManager extends EventTarget {
    readonly #spawn: () => Worker
    #worker: Worker | null = null
    #running = false

    constructor(spawn: () => Worker) {
        super()
        this.#spawn = spawn
    }

    /** Whether the worker was told to run and not since told to stop. */
    get running(): boolean {
        return this.#running
    }

    /** Create the worker (if needed) and begin streaming from `baseUrl`. */
    start(baseUrl: string): void {
        if (this.#running) return

        // The worker outlives a stop, so a restart reuses the thread.
        this.#worker ??= this.#spawn()
        this.#worker.onmessage = (event: MessageEvent<WorkerToPageMessage>) => {
            this.dispatchEvent(new CustomEvent(NOTIFY_EVENT, { detail: event.data }))
        }

        this.#running = true
        this.#post({ type: 'start', baseUrl })
        this.dispatchEvent(new Event(STATE_EVENT))
    }

    /** Abort the worker's loop, keeping the worker alive for reuse. */
    stop(): void {
        if (!this.#running) return

        this.#running = false
        this.#post({ type: 'stop' })
        this.dispatchEvent(new Event(STATE_EVENT))
    }

    /** Tear the worker down entirely. */
    terminate(): void {
        this.#worker?.terminate()
        this.#worker = null

        if (!this.#running) return

        this.#running = false
        this.dispatchEvent(new Event(STATE_EVENT))
    }

    /** Subscribe to worker notifications. Returns an unsubscribe function. */
    subscribe(handler: (message: WorkerToPageMessage) => void): () => void {
        const listener = (event: Event) => {
            handler((event as CustomEvent<WorkerToPageMessage>).detail)
        }
        this.addEventListener(NOTIFY_EVENT, listener)
        return () => this.removeEventListener(NOTIFY_EVENT, listener)
    }

    /** Subscribe to `running` changes. Shaped for `useSyncExternalStore`. */
    watch(listener: () => void): () => void {
        this.addEventListener(STATE_EVENT, listener)
        return () => this.removeEventListener(STATE_EVENT, listener)
    }

    #post(message: PageToWorkerMessage): void {
        this.#worker?.postMessage(message)
    }
}
