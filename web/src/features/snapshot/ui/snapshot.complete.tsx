import type { FC } from 'react'

import { CompleteInfo } from './snapshot.complete.info'
import { CompleteSinks } from './snapshot.complete.sinks'
import { CompleteStats } from './snapshot.complete.stat'
import { CompleteTables } from './snapshot.complete.tables'

export const CompleteView: FC = () => (
    <>
        <CompleteInfo />
        <CompleteStats />
        <div className="flex flex-col gap-4 lg:flex-row">
            <CompleteTables />
            <CompleteSinks />
        </div>
    </>
)
