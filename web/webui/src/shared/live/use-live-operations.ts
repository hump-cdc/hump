import { useEffect, useRef } from 'react'

import { liveStream } from './stream'
import type { LiveOperationsListener } from './types'

/**
 * Subscribes to live delivery batches for the lifetime of the component. `handler` is kept in a
 * ref, so an inline callback does not tear the subscription down on every render.
 */
export const useLiveOperations = (handler: LiveOperationsListener): void => {
    const handlerRef = useRef(handler)
    handlerRef.current = handler

    useEffect(() => liveStream.subscribe((batch) => handlerRef.current(batch)), [])
}
