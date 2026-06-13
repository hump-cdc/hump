import type { FC } from 'react'

import { StartInfo } from './snapshot.start.info'
import { StartTables } from './snapshot.start.tables'

export const StartView: FC = () => (
    <>
        <StartInfo />
        <StartTables />
    </>
)
