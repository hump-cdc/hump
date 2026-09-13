import type { FC } from 'react'

import { Delivered } from '@/entities/dashboard/delivered'
import { Failed } from '@/entities/dashboard/failed'
import { HeapInUse } from '@/entities/dashboard/heap-in-use'
import { ReplicationLag } from '@/entities/dashboard/replication-lag'
import { SinkDelivery } from '@/entities/dashboard/sink-delivery'
import { SinkLatency } from '@/entities/dashboard/sink-latency'
import { Sinks } from '@/entities/dashboard/sinks'
import { Tables } from '@/entities/dashboard/tables'
import { Throughput } from '@/entities/dashboard/throughput'

export const Dashboard: FC = () => (
    <div className="flex flex-1 flex-col gap-4 p-6 max-w-6xl">
        <ReplicationLag />
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2"></div>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Delivered />
            <Failed />
        </div>
        <Throughput />
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
