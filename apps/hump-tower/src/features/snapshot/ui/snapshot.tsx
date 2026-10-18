import { type AnimationEvent, type FC, type ReactNode, useState } from 'react'

import { cn, SwitchViewValueContext, useSwitchAnimation } from '@/shared/utils'

import { useSnapshotState } from '../model/snapshot'
import type { SnapshotState } from '../model/types'
import { DoneActionBar, DoneView } from './done'
import { IdleActionBar, IdleView } from './idle'
import { ProcessingActionBar, ProcessingView } from './processing'

const views: Record<SnapshotState['state'], ReactNode> = {
    idle: <IdleView />,
    processing: <ProcessingView />,
    done: <DoneView />,
}

const actionBars: Record<SnapshotState['state'], ReactNode> = {
    idle: <IdleActionBar />,
    processing: <ProcessingActionBar />,
    done: <DoneActionBar />,
}

export const Snapshot: FC = () => {
    const { data: state } = useSnapshotState()
    if (!state) return null

    return <SnapshotStates state={state} />
}

/**
 * Snapshot states stacked in a single grid cell, so switching state cross-fades the outgoing view over the incoming one.
 */
const SnapshotStates: FC<{ state: SnapshotState }> = ({ state }) => {
    const { current, leaving, views: states, onLeaveEnd } = useSwitchAnimation(state.state)

    const [shown, setShown] = useState<Partial<Record<SnapshotState['state'], SnapshotState>>>({})
    if (shown[state.state] !== state) {
        setShown({ ...shown, [state.state]: state })
    }

    return (
        <div className="isolate grid flex-1">
            {states.map((view) => {
                const isLeaving = view !== current

                return (
                    <div
                        key={view}
                        inert={isLeaving}
                        className={cn(
                            'col-start-1 row-start-1 flex flex-col gap-4 duration-400 ease-in-out',
                            leaving !== null &&
                                cn(
                                    'mix-blend-plus-lighter',
                                    isLeaving ? 'fade-out-0 animate-out fill-mode-forwards' : 'fade-in-0 animate-in',
                                ),
                        )}
                        onAnimationEnd={(event: AnimationEvent) => {
                            if (isLeaving && event.target === event.currentTarget) {
                                onLeaveEnd()
                            }
                        }}
                    >
                        <SwitchViewValueContext value={view === state.state ? state : shown[view]}>
                            {views[view]}
                            {actionBars[view]}
                        </SwitchViewValueContext>
                    </div>
                )
            })}
        </div>
    )
}
