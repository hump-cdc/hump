import type { FC } from 'react'

import type { SelectedItem } from '../model/types'
import { DetailedSinkDetails } from './details.detailed-sink'
import { EdgeDetails } from './details.edge'
import { SinkDetails } from './details.sink'
import { TableDetails } from './details.table'
import { TargetDetails } from './details.target'

/** Renders the identity card + metrics for the current selection, one component per kind. */
export const InspectorDetails: FC<{ selected: SelectedItem }> = ({ selected }) => {
    switch (selected.kind) {
        case 'edge':
            return <EdgeDetails selected={selected} />
        case 'table':
            return <TableDetails selected={selected} />
        case 'sink':
            return <SinkDetails selected={selected} />
        case 'detailed-sink':
            return <DetailedSinkDetails selected={selected} />
        case 'target':
            return <TargetDetails selected={selected} />
    }
}
