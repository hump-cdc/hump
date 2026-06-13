import type { FC } from 'react'

import { AsyncQueueDepth } from '@/entities/stats/async-queue-depth'
import { CommitLag } from '@/entities/stats/commit-lag'
import { FailRate } from '@/entities/stats/fail-rate'
import { GcPause } from '@/entities/stats/gc-pause'
import { Goroutines } from '@/entities/stats/goroutines'
import { HeapInUse } from '@/entities/stats/heap-in-use'
import { OpenFds } from '@/entities/stats/open-fds'
import { ReplicationLag } from '@/entities/stats/replication-lag'
import { RowsPerSec } from '@/entities/stats/rows-per-sec'
import { SenderLag } from '@/entities/stats/sender-lag'
import { SinkDelivery } from '@/entities/stats/sink-delivery'
import { SinkLatency } from '@/entities/stats/sink-latency'
import { Throughput } from '@/entities/stats/throughput'

export const Stats: FC = () => (
    <div className="flex flex-1 flex-col gap-4 p-6 max-w-6xl">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <CommitLag />
            <SenderLag />
            <RowsPerSec />
            <FailRate />
        </div>

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <ReplicationLag />
            <Throughput />
            <SinkDelivery />
            <SinkLatency />
        </div>

        <AsyncQueueDepth />

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <HeapInUse />
            <GcPause />
            <Goroutines />
            <OpenFds />
        </div>
    </div>
)
