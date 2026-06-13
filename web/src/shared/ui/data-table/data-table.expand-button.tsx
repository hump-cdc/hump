import type { FC } from 'react'

import type { CustomCellRendererProps } from 'ag-grid-react'
import { ChevronDown, ChevronRight } from 'lucide-react'

import { cn } from '@/shared/utils'

import type { ExpandContext, InternalRow } from './data-table.helpers.expandable'

/**
 * Leading-column cell renderer that toggles row expansion. Renders a chevron
 * only for expandable master rows; other rows render nothing.
 */
export const ExpandButton: FC<CustomCellRendererProps> = ({ data, context }) => {
    const row = data as InternalRow<unknown> | undefined
    if (row?.kind !== 'master' || !row.expandable) {
        return null
    }

    const { toggle } = context as ExpandContext
    const Icon = row.expanded ? ChevronDown : ChevronRight

    return (
        <button
            type="button"
            aria-label={row.expanded ? 'Collapse row' : 'Expand row'}
            aria-expanded={row.expanded}
            onClick={() => toggle(row.id)}
            className="flex size-full items-center justify-center text-ink-tertiary hover:text-ink-muted"
        >
            <Icon className={cn('size-3.75', row.expanded && 'text-ink-muted')} />
        </button>
    )
}
