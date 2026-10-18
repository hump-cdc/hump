import { mock } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'

import * as actual from './stream'
import type {
    LiveOperationsBatch,
    LiveOperationsListener,
    LiveSnapshot,
    LiveStatus,
    LiveStatusListener,
    Unsubscribe,
} from './types'

/** A `LiveStream` driven by hand, plus what a test needs to look inside it. */
export type FakeLiveStream = actual.LiveStream & {
    emit: (batch: LiveOperationsBatch) => void
    setStatus: (status: LiveStatus) => void
    setSnapshot: (snapshot: LiveSnapshot | undefined) => void
    reconnect: () => void
    /** Holds taken through `retain` and not released yet. */
    holds: () => number
    /** Delivery listeners currently subscribed. */
    listeners: () => number
    reset: () => void
}

/**
 * The stream the hooks and `LiveUpdates` see in tests, in place of the app-wide `liveStream`.
 * One instance for every test file: `mock.module` is process-wide, so each test resets it.
 */
export const fakeLiveStream: FakeLiveStream = (() => {
    const operations = new Set<LiveOperationsListener>()
    const statuses = new Set<LiveStatusListener>()
    const snapshots = new Set<() => void>()
    const reconnects = new Set<() => void>()
    let status: LiveStatus = 'idle'
    let snapshot: LiveSnapshot | undefined
    let holds = 0

    const add = <T>(set: Set<T>, listener: T): Unsubscribe => {
        set.add(listener)
        return () => {
            set.delete(listener)
        }
    }

    return {
        retain: () => {
            holds += 1
            return () => {
                holds -= 1
            }
        },
        stop: () => {
            holds = 0
        },
        subscribe: (listener) => add(operations, listener),
        subscribeStatus: (listener) => add(statuses, listener),
        subscribeReconnect: (listener) => add(reconnects, listener),
        subscribeSnapshot: (listener) => add(snapshots, listener),
        getStatus: () => status,
        getSnapshot: () => snapshot,
        emit: (batch) => {
            for (const listener of operations) listener(batch)
        },
        setStatus: (next) => {
            status = next
            for (const listener of statuses) listener()
        },
        setSnapshot: (next) => {
            snapshot = next
            for (const listener of snapshots) listener()
        },
        reconnect: () => {
            for (const listener of reconnects) listener()
        },
        holds: () => holds,
        listeners: () => operations.size,
        reset: () => {
            operations.clear()
            statuses.clear()
            snapshots.clear()
            reconnects.clear()
            status = 'idle'
            snapshot = undefined
            holds = 0
        },
    }
})()

// Everything but the singleton stays real, so `stream.test.ts` is unaffected by the swap.
mock.module('./stream', () => ({ ...actual, liveStream: fakeLiveStream }))

/**
 * Installs the DOM globals React needs. Paired with `unregisterDom` per test file rather than
 * preloaded, so the files that talk to a real server keep the runtime's own `fetch`.
 */
export const registerDom = (): void => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
}

/** Removes the DOM globals again. */
export const unregisterDom = async (): Promise<void> => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
}
