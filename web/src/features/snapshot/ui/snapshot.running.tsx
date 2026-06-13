import type { FC } from 'react'

import { RunningProgress } from './snapshot.running.progress'
import { RunningStats } from './snapshot.running.stat'
import { RunningTables } from './snapshot.running.tables'

export const RunningView: FC = () => (
    <>
        <RunningProgress />
        <RunningStats />
        <RunningTables />
    </>
)
