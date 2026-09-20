import { useSyncExternalStore } from 'react'

import { liveStream } from './stream'
import type { LiveStatus } from './types'

/** Current connection status. Only components that call this re-render on a transition. */
export const useLiveStatus = (): LiveStatus =>
    useSyncExternalStore(liveStream.subscribeStatus, liveStream.getStatus, liveStream.getStatus)
