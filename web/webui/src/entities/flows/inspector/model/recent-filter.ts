import type { OperationsFilterKey } from '@/shared/state'

import type { SelectedItem } from './types'

/** The operations log filter that lists the deliveries of the selected node or edge. */
export const toRecentFilter = (selected: SelectedItem): OperationsFilterKey => {
    switch (selected.kind) {
        case 'table':
            return { tables: [selected.name] }
        case 'sink':
            return { sinks: [selected.type], hosts: [selected.host] }
        case 'target':
            return { sinks: [selected.type], hosts: [selected.host], targets: [selected.name] }
        case 'edge':
            return {
                tables: [selected.table],
                sinks: [selected.type],
                hosts: [selected.host],
                ...(selected.target === undefined ? {} : { targets: [selected.target] }),
            }
    }
}
