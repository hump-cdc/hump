import type { OperationEventModel, SnapshotEventModel } from '@/shared/api'

/** One finished delivery, reduced to where it went. */
export type LiveOperation = OperationEventModel

/** Deliveries of one backend window, oldest first. An empty batch never reaches listeners. */
export type LiveOperationsBatch = readonly LiveOperation[]

/**
 * `connecting` lasts until the first event, which the server sends as soon as it accepts the
 * connection. Healthy states are `connecting` and `live`; only `reconnecting` is degraded.
 */
export type LiveStatus = 'idle' | 'connecting' | 'live' | 'reconnecting'

/** The snapshot run reduced to its state and overall progress in whole percent. */
export type LiveSnapshot = SnapshotEventModel

export type LiveOperationsListener = (batch: LiveOperationsBatch) => void

/** Notified on every status transition; reads the new value through `getStatus`. */
export type LiveStatusListener = () => void

/** Notified when the snapshot value changes; reads the new value through `getSnapshot`. */
export type LiveSnapshotListener = () => void

/** Notified before each retry, because events in the gap are lost for good. */
export type LiveReconnectListener = () => void

export type Unsubscribe = () => void

/** Drops one hold on the connection; the stream stops once the last one is released. */
export type LiveRelease = () => void
