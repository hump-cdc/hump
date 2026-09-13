import type { FC } from 'react'

import { Card, CardContent, CardHeader } from '@/shared/ui/card'

export const StartInfo: FC = () => (
    <Card>
        <CardHeader>
            <span className="text-md font-semibold text-ink">How does it work</span>
        </CardHeader>
        <CardContent className="p-4">
            <p className="max-w-3xl text-base leading-normal text-ink-subtle">
                Hump normally streams live changes. A snapshot re-reads the selected tables in full and re-emits every
                row to all connected sinks — use it to rebuild a sink that was reset or to seed a newly added consumer.
            </p>
        </CardContent>
    </Card>
)
