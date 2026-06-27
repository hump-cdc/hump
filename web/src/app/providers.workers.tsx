import { type FC, type PropsWithChildren, useEffect } from 'react'

import { liveStream } from '@/shared/workers/live-stream'

export const WorkersProvider: FC<PropsWithChildren> = ({ children }) => {
    // Adopt whatever the user last chose rather than starting unconditionally, so a tab
    // opened while the feed is paused comes up paused too.
    useEffect(() => liveStream.attach(), [])

    return children
}
