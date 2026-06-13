import type { FC } from 'react'

import { FlowInspector } from '@/entities/flows/inspector'
import { FlowsTopology } from '@/entities/flows/topology'

export const Flows: FC = () => <FlowsTopology overlay={<FlowInspector />} />
