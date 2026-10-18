import { useQueryClient } from '@tanstack/react-query'

import type { FC } from 'react'
import { useEffect } from 'react'

import { invalidateForLiveEvent, invalidateForLiveReconnect } from '@/shared/state'

import { liveStream } from './stream'

/**
 * Owns the app-wide live connection and refreshes the queries a delivery makes stale. Renders
 * nothing and holds no state, so an event can never re-render the tree through this component.
 */
export const LiveUpdates: FC = () => {
    const queryClient = useQueryClient()

    useEffect(() => liveStream.retain(), [])
    useEffect(() => liveStream.subscribe(() => invalidateForLiveEvent(queryClient)), [queryClient])
    useEffect(() => liveStream.subscribeReconnect(() => invalidateForLiveReconnect(queryClient)), [queryClient])

    return null
}
