import type { FC } from 'react'

import { AsyncQueue } from '@/entities/stats/async-queue'
import { FailRate } from '@/entities/stats/fail-rate'
import { HeapInUse } from '@/entities/stats/heap-in-use'
import { ReplicationLag } from '@/entities/stats/replication-lag'
import { SinkDelivery } from '@/entities/stats/sink-delivery'
import { SinkLatency } from '@/entities/stats/sink-latency'
import { Throughput } from '@/entities/stats/throughput'

export const Stats: FC = () => (
    <div className="flex flex-1 flex-col gap-4 p-6 max-w-6xl">
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <ReplicationLag />
            <Throughput />
            <SinkDelivery />
            <SinkLatency />
        </div>

        <AsyncQueue />

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <FailRate />
            <HeapInUse />
        </div>
    </div>
)
