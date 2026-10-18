import type { FC } from 'react'

import { CircleCheck, CircleX, RotateCcw } from 'lucide-react'

import { Badge } from '@/shared/ui/badge'
import { cn } from '@/shared/utils'

type OperationStatusBadgeProps = {
    status: 'delivered' | 'failed'
    attempts: number
    /** `full` is a labeled badge, `compact` a bare delivered/failed icon. */
    variant?: 'full' | 'compact'
    /** Spells the attempt count out in the `full` label, for layouts with no separate attempts line. */
    withAttempts?: boolean
    className?: string
}

/** Terminal delivery outcome; in the `full` variant a delivery that needed retries shows amber. */
export const OperationStatusBadge: FC<OperationStatusBadgeProps> = ({
    status,
    attempts,
    variant = 'full',
    withAttempts,
    className,
}) => {
    if (variant === 'compact') {
        return status === 'failed' ? (
            <CircleX aria-label="Failed" className={cn('size-3.75 text-danger', className)} />
        ) : (
            <CircleCheck aria-label="Delivered" className={cn('size-3.75 text-success', className)} />
        )
    }

    const retried = attempts > 1
    const suffix = withAttempts ? ` after ${attempts} attempts` : ' after retry'

    if (status === 'failed') {
        return (
            <Badge variant="failed" className={className}>
                <CircleX className="size-3" />
                {retried ? `Failed${suffix}` : 'Failed'}
            </Badge>
        )
    }
    if (retried) {
        return (
            <Badge variant="lag" className={className}>
                <RotateCcw className="size-3" />
                {`Delivered${suffix}`}
            </Badge>
        )
    }
    return (
        <Badge variant="healthy" className={className}>
            <CircleCheck className="size-3" />
            Delivered
        </Badge>
    )
}
