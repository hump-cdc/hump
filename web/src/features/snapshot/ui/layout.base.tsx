import type { FC, PropsWithChildren, ReactNode } from 'react'

export const SnapshotBase: FC<
    PropsWithChildren<{
        action?: ReactNode
    }>
> = ({ children, action }) => (
    <div className="flex flex-1 flex-col gap-4">
        {children}
        {action}
    </div>
)
