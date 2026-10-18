import type { FC } from 'react'

import { IdleInfo } from './info'
import { Tables } from './tables'

export { IdleActionBar } from './action-bar'

export const IdleView: FC = () => (
    <>
        <IdleInfo />
        <Tables />
    </>
)
