import type { FC } from 'react'

import { SourceViewer } from '@/entities/config/source-viewer'

export const Configuration: FC = () => (
    <div className="flex h-[calc(100svh-3.5rem)] flex-col p-6">
        <SourceViewer />
    </div>
)
