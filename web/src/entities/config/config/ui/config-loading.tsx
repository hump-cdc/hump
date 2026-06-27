import type { FC } from 'react'

import { LoaderCircle } from 'lucide-react'

export const ConfigLoading: FC = () => (
    <div className="flex h-screen items-center justify-center p-6">
        <LoaderCircle className="size-6 animate-spin text-ink-subtle" />
    </div>
)
