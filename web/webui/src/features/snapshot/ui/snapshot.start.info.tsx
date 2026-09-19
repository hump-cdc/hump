import type { FC } from 'react'

import { Card, CardContent, CardHeader } from '@/shared/ui/card'

export const StartInfo: FC = () => (
    <Card>
        <CardHeader>
            <span className="font-semibold text-ink text-md">How does it work</span>
        </CardHeader>
        <CardContent className="p-4">
            <p className="max-w-3xl text-base text-ink-subtle leading-normal">
                Hump normally streams live changes. A snapshot re-reads the selected tables in full and re-emits every
                row to all connected sinks — use it to rebuild a sink that was reset or to seed a newly added consumer.
            </p>
        </CardContent>
    </Card>
)
