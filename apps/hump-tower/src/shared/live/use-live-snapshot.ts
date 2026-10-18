import { useSyncExternalStore } from 'react'

import { liveStream } from './stream'
import type { LiveSnapshot } from './types'

/** Snapshot state reported by the live feed; `undefined` until the server sends one. */
export const useLiveSnapshot = (): LiveSnapshot | undefined =>
    useSyncExternalStore(liveStream.subscribeSnapshot, liveStream.getSnapshot, liveStream.getSnapshot)
