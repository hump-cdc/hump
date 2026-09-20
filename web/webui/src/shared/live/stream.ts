import { Live, type OperationsEventModel } from '@/shared/api'
import { sleep } from '@/shared/utils'

import { createBackoff, STABLE_ATTEMPT_MS } from './backoff'
import type {
    LiveOperationsListener,
    LiveReconnectListener,
    LiveRelease,
    LiveStatus,
    LiveStatusListener,
    Unsubscribe,
} from './types'

/** How long a released connection stays open, so a StrictMode remount reuses it. */
const LINGER_MS = 1_000

/** Opens one attempt at the live feed. Reconnecting is the caller's job. */
export type LiveStreamOpen = (signal: AbortSignal) => Promise<AsyncIterable<OperationsEventModel>>

/** Injection points for tests; every one falls back to the production implementation. */
export type LiveStreamDeps = {
    open?: LiveStreamOpen
    sleep?: (ms: number, signal?: AbortSignal) => Promise<void>
    now?: () => number
    lingerMs?: number
}

export type LiveStream = {
    /** Holds the connection open; call the returned release on cleanup. Reference-counted. */
    retain: () => LiveRelease
    /** Closes the connection and drops every hold. For teardown — HMR and tests — not for consumers. */
    stop: () => void
    subscribe: (listener: LiveOperationsListener) => Unsubscribe
    subscribeStatus: (listener: LiveStatusListener) => Unsubscribe
    subscribeReconnect: (listener: LiveReconnectListener) => Unsubscribe
    getStatus: () => LiveStatus
}

/**
 * Builds a live feed that reconnects on its own and fans batches out to listeners. Holds no
 * React state: listeners are called imperatively, so an event costs no render by itself.
 */
export const createLiveStream = (deps: LiveStreamDeps = {}): LiveStream => {
    const open = deps.open ?? openLiveStream
    const wait = deps.sleep ?? sleep
    const now = deps.now ?? Date.now
    const lingerMs = deps.lingerMs ?? LINGER_MS

    const operationsListeners = new Set<LiveOperationsListener>()
    const statusListeners = new Set<LiveStatusListener>()
    const reconnectListeners = new Set<LiveReconnectListener>()

    let status: LiveStatus = 'idle'
    let controller: AbortController | undefined
    let holds = 0
    let stopTimer: ReturnType<typeof setTimeout> | undefined

    const setStatus = (next: LiveStatus): void => {
        if (next === status) {
            return
        }

        status = next
        notify(statusListeners, (listener) => listener())
    }

    const run = async (signal: AbortSignal): Promise<void> => {
        const backoff = createBackoff()
        let attempt = 0

        while (!signal.aborted) {
            attempt += 1
            if (attempt > 1) {
                notify(reconnectListeners, (listener) => listener())
            }
            setStatus('connecting')

            const startedAt = now()

            try {
                for await (const event of await open(signal)) {
                    const operations = event.operations ?? []
                    if (operations.length === 0) {
                        continue
                    }

                    setStatus('live')
                    notify(operationsListeners, (listener) => listener(operations))
                }
            } catch (error) {
                if (!signal.aborted) {
                    console.warn('[live] stream attempt failed', error)
                }
            }

            // A clean end of the generator is not terminal either: the server restarting looks
            // exactly like this, and the built-in transport would stop for good.
            if (signal.aborted) {
                break
            }

            if (now() - startedAt >= STABLE_ATTEMPT_MS) {
                backoff.reset()
            }

            setStatus('reconnecting')
            await wait(backoff.next(), signal)
        }
    }

    const start = (): void => {
        if (controller) {
            return
        }

        const own = new AbortController()
        controller = own

        void run(own.signal).finally(() => {
            if (controller === own) {
                controller = undefined
                setStatus('idle')
            }
        })
    }

    const stop = (): void => {
        clearTimeout(stopTimer)
        stopTimer = undefined
        holds = 0
        controller?.abort()
        controller = undefined
        setStatus('idle')
    }

    return {
        retain: () => {
            holds += 1
            clearTimeout(stopTimer)
            stopTimer = undefined
            start()

            let released = false

            return () => {
                if (released) {
                    return
                }

                released = true
                holds -= 1
                if (holds > 0) {
                    return
                }

                stopTimer = setTimeout(() => {
                    stopTimer = undefined
                    if (holds === 0) {
                        stop()
                    }
                }, lingerMs)
            }
        },
        stop,
        subscribe: (listener) => subscribeTo(operationsListeners, listener),
        subscribeStatus: (listener) => subscribeTo(statusListeners, listener),
        subscribeReconnect: (listener) => subscribeTo(reconnectListeners, listener),
        getStatus: () => status,
    }
}

/**
 * Opens `/api/live` for a single attempt. The built-in retry loop never resets its backoff,
 * stops for good on a clean stream end, and sleeps the whole backoff before noticing an abort,
 * so reconnecting is owned here instead.
 */
const openLiveStream: LiveStreamOpen = async (signal) => {
    const { stream } = await Live.stream({
        signal,
        sseMaxRetryAttempts: 1,
        // The transport swallows a failed event and ends the iteration instead of throwing, so
        // without this a payload the schema rejects would reconnect in silence.
        onSseError: (error) => {
            if (!signal.aborted) {
                console.warn('[live] stream error', error)
            }
        },
    })

    return stream
}

/** Adds `listener` to `listeners` and returns the matching removal. */
const subscribeTo = <T>(listeners: Set<T>, listener: T): Unsubscribe => {
    listeners.add(listener)

    return () => {
        listeners.delete(listener)
    }
}

/**
 * Calls every listener over a copy of the set, so unsubscribing from inside one cannot break
 * the iteration, and isolates throws, so a broken listener cannot take the stream down.
 */
const notify = <T>(listeners: Set<T>, call: (listener: T) => void): void => {
    for (const listener of [...listeners]) {
        try {
            call(listener)
        } catch (error) {
            console.error('[live] listener failed', error)
        }
    }
}

/** App-wide live feed. */
export const liveStream = createLiveStream()

// Editing this module in dev would otherwise leave the previous singleton's connection open.
import.meta.webpackHot?.dispose(() => {
    liveStream.stop()
})
