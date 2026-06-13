import type { FC, ReactNode } from 'react'

import type { SnapshotState } from '../model/types'
import { useSnapshot } from '../model/use-snapshot'
import { SnapshotBase } from './layout.base'
import { CompleteView } from './snapshot.complete'
import { CompleteActionBar } from './snapshot.complete.action-bar'
import { RunningView } from './snapshot.running'
import { RunningActionBar } from './snapshot.running.action-bar'
import { StartView } from './snapshot.start'
import { StartActionBar } from './snapshot.start.action-bar'

const views: Record<SnapshotState, ReactNode> = {
    initial: <StartView />,
    processing: <RunningView />,
    finished: <CompleteView />,
}

const actionBars: Record<SnapshotState, ReactNode> = {
    initial: <StartActionBar />,
    processing: <RunningActionBar />,
    finished: <CompleteActionBar />,
}

export const Snapshot: FC = () => {
    const state = useSnapshot((store) => store.state)
    return <SnapshotBase action={actionBars[state]}>{views[state]}</SnapshotBase>
}
