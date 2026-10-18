import type { ComponentProps, FC } from 'react'

import { Badge } from '@/shared/ui/badge'
import { cn } from '@/shared/utils'

type Operation = 'insert' | 'update' | 'delete' | 'truncate'

/** Badge color per row operation. */
const OPERATION_VARIANTS = {
    insert: 'healthy',
    update: 'active',
    delete: 'failed',
    truncate: 'lag',
} as const satisfies Record<Operation, ComponentProps<typeof Badge>['variant']>

type OperationBadgeProps = Omit<ComponentProps<typeof Badge>, 'variant' | 'children'> & {
    op: Operation
}

export const OperationBadge: FC<OperationBadgeProps> = ({ className, op, ...props }) => (
    <Badge className={cn('px-1 uppercase leading-4', className)} variant={OPERATION_VARIANTS[op]} {...props}>
        {op}
    </Badge>
)
