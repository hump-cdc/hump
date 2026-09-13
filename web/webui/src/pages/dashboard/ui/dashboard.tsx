import type { FC } from 'react'

import { AsyncQueue } from '@/entities/dashboard/async-queue'
import { CommitLag } from '@/entities/dashboard/commit-lag'
import { Delivered } from '@/entities/dashboard/delivered'
import { Failed } from '@/entities/dashboard/failed'
import { SenderLag } from '@/entities/dashboard/sender-lag'
import { Sinks } from '@/entities/dashboard/sinks'
import { Tables } from '@/entities/dashboard/tables'
import { Throughput } from '@/entities/dashboard/throughput'

export const Dashboard: FC = () => (
    <div className="flex flex-1 flex-col gap-4 p-6 max-w-6xl">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <CommitLag />
            <SenderLag />
            <Throughput />
            <AsyncQueue />
        </div>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Delivered />
            <Failed />
        </div>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Tables />
            <Sinks />
        </div>
    </div>
)
