import type { FC } from 'react'

import { Snapshot } from '@/features/snapshot'

export const SnapshotPage: FC = () => (
    <div className="flex flex-1 flex-col gap-4 p-6 max-w-6xl">
        <Snapshot />
    </div>
)
