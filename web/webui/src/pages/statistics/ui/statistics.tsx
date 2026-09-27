import type { FC } from 'react'

import { Delivered } from '@/entities/statistics/delivered'
import { Failed } from '@/entities/statistics/failed'
import { HeapInUse } from '@/entities/statistics/heap-in-use'
import { ReplicationLag } from '@/entities/statistics/replication-lag'
import { SinkDelivery } from '@/entities/statistics/sink-delivery'
import { SinkLatency } from '@/entities/statistics/sink-latency'
import { Sinks } from '@/entities/statistics/sinks'
import { Tables } from '@/entities/statistics/tables'
import { Throughput } from '@/entities/statistics/throughput'

export const Statistics: FC = () => (
    <div className="flex max-w-6xl flex-1 flex-col gap-4 p-6">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Delivered />
            <Failed />
        </div>
        <Throughput />
        <ReplicationLag />
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <SinkDelivery />
            <SinkLatency />
        </div>
        <HeapInUse />
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Tables />
            <Sinks />
        </div>
    </div>
)
