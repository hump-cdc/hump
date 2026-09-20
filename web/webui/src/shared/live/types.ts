import type { OperationEventModel } from '@/shared/api'

/** One finished delivery, reduced to where it went. */
export type LiveOperation = OperationEventModel

/** Deliveries of one backend window, oldest first. An empty batch never reaches listeners. */
export type LiveOperationsBatch = readonly LiveOperation[]

/**
 * `connecting` covers both waiting for response headers and waiting for the first batch: the
 * server writes nothing until a delivery happens, so an idle pipeline stays `connecting`.
 * Healthy states are `connecting` and `live`; only `reconnecting` is degraded.
 */
export type LiveStatus = 'idle' | 'connecting' | 'live' | 'reconnecting'

export type LiveOperationsListener = (batch: LiveOperationsBatch) => void

/** Notified on every status transition; reads the new value through `getStatus`. */
export type LiveStatusListener = () => void

/** Notified before each retry, because events in the gap are lost for good. */
export type LiveReconnectListener = () => void

export type Unsubscribe = () => void

/** Drops one hold on the connection; the stream stops once the last one is released. */
export type LiveRelease = () => void
