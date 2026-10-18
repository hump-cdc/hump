import type { FC } from 'react'

import { Snapshot } from '@/features/snapshot'

export const SnapshotPage: FC = () => (
    <div className="flex max-w-6xl flex-1 flex-col gap-4 p-6">
        <Snapshot />
    </div>
)
