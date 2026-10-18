import { Link } from '@tanstack/react-router'

import type { ComponentProps, FC } from 'react'

import { useLiveSnapshot } from '@/shared/live'
import { Badge } from '@/shared/ui/badge'
import { NumberFlow } from '@/shared/ui/number-flow'
import { cn } from '@/shared/utils'

type BadgeProps = ComponentProps<typeof Badge>

export const SnapshotStatus: FC<Omit<BadgeProps, 'variant' | 'children'>> = ({ className, ...props }) => {
    const snapshot = useLiveSnapshot()

    if (snapshot?.state !== 'processing') {
        return null
    }

    return (
        <Link to="/snapshot" className="inline-flex">
            <Badge variant="active" dot dotAnimated className={cn('cursor-pointer px-3', className)} {...props}>
                Snapshot <NumberFlow value={snapshot.progress} kind="percent" />
            </Badge>
        </Link>
    )
}
