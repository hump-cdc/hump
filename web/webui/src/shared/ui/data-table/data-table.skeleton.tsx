import type { FC } from 'react'

export const Skeleton: FC = () => (
    <div className="flex h-full items-center">
        <span className="block h-3 w-2/3 animate-pulse rounded bg-surface-3" />
    </div>
)
