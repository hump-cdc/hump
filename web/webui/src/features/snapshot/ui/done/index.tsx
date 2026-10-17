import type { FC } from 'react'

import { DoneInfo } from './info'
import { DoneSinks } from './sinks'
import { DoneStats } from './stat'
import { DoneTables } from './tables'

export { DoneActionBar } from './action-bar'

export const DoneView: FC = () => (
    <>
        <DoneInfo />
        <DoneStats />
        <div className="flex flex-col gap-4 xl:flex-row">
            <DoneTables />
            <DoneSinks />
        </div>
    </>
)
