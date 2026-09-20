import type { InvalidateQueryFilters, QueryClient } from '@tanstack/react-query'

import { keys } from './keys'

/** Queries a batch of deliveries makes stale. */
export const LIVE_EVENT_QUERIES: InvalidateQueryFilters[] = [{ queryKey: keys.prefixes.metrics() }]

/** The same queries after a gap in the stream, refetched even while unmounted: events were missed. */
export const LIVE_RECONNECT_QUERIES: InvalidateQueryFilters[] = [
    { queryKey: keys.prefixes.metrics(), refetchType: 'all' },
]

/** Marks stale everything a live batch affects. */
export const invalidateForLiveEvent = (client: QueryClient): void => invalidate(client, LIVE_EVENT_QUERIES)

/** Marks stale everything the stream may have missed while it was down. */
export const invalidateForLiveReconnect = (client: QueryClient): void => invalidate(client, LIVE_RECONNECT_QUERIES)

/**
 * Invalidates every filter without cancelling in-flight refetches. `cancelRefetch` defaults to
 * `true`, which at two events per second would restart any slower request forever.
 */
const invalidate = (client: QueryClient, filters: InvalidateQueryFilters[]): void => {
    for (const filter of filters) {
        void client.invalidateQueries(filter, { cancelRefetch: false })
    }
}
