import type { FC } from 'react'

import { Card, CardContent, CardHeader } from '@/shared/ui/card'
import { Mark } from '@/shared/ui/mark'

export const IdleInfo: FC = () => (
    <Card>
        <CardHeader>When to use a snapshot</CardHeader>
        <CardContent className="flex flex-col gap-3 p-4 text-base text-ink-subtle leading-normal">
            <p>
                A snapshot re-reads the selected tables and sends their current rows to all connected sinks. It is
                useful when you need to:
            </p>
            <ul className="flex list-disc flex-col gap-1 pl-5">
                <li>
                    <span className="font-semibold">Rebuild a sink</span> after it was reset
                </li>
                <li>
                    <span className="font-semibold">Seed new consumers</span> after the sink has been drained
                </li>
            </ul>
            <p>
                Add an optional <Mark>WHERE</Mark> condition per table to snapshot only matching rows.
            </p>
        </CardContent>
    </Card>
)
