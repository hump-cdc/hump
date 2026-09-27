import type { ComponentProps, FC } from 'react'

import { type LiveStatus as LiveStreamStatus, useLiveStatus } from '@/shared/live'
import { Badge } from '@/shared/ui/badge'

type BadgeProps = ComponentProps<typeof Badge>

const STATUS_BADGES: Record<LiveStreamStatus, { variant: BadgeProps['variant']; label: string }> = {
    idle: { variant: 'failed', label: 'offline' },
    connecting: { variant: 'lag', label: 'connecting' },
    live: { variant: 'healthy', label: 'live' },
    reconnecting: { variant: 'failed', label: 'offline' },
}

export const LiveStatus: FC<Omit<BadgeProps, 'variant' | 'children'>> = (props) => {
    const status = useLiveStatus()
    const badge = STATUS_BADGES[status]

    return (
        <Badge variant={badge.variant} dot {...props}>
            {badge.label}
        </Badge>
    )
}
