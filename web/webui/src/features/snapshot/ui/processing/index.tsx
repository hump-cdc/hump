import type { FC } from 'react'

import { ProcessingOverall } from './overall'
import { ProcessingStats } from './stat'
import { ProcessingTables } from './tables'

export { ProcessingActionBar } from './action-bar'

export const ProcessingView: FC = () => (
    <>
        <ProcessingOverall />
        <ProcessingStats />
        <ProcessingTables />
    </>
)
