import type { SelectedItem } from './types'

/** Search params of `/operations` that preset its filter to a topology item. */
export type OperationsSearch = {
    table?: string
    sink?: string
    target?: string
}

/**
 * Search params that open `/operations` filtered to the selected node or edge. The operations
 * filter has no host facet, so a sink narrows by its type only.
 */
export const toOperationsSearch = (selected: SelectedItem): OperationsSearch => {
    switch (selected.kind) {
        case 'table':
            return { table: selected.name }
        case 'sink':
            return { sink: selected.type }
        case 'target':
            return { sink: selected.type, target: selected.name }
        case 'edge':
            return {
                table: selected.table,
                sink: selected.type,
                ...(selected.target === undefined ? {} : { target: selected.target }),
            }
    }
}
