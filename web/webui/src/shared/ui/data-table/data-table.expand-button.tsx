import type { FC } from 'react'

import { ChevronDown, ChevronRight } from 'lucide-react'

import { cn } from '@/shared/utils'

type ExpandButtonProps = {
    isExpanded: boolean
    onToggle: () => void
}

/** Chevron in the leading column that toggles a row's detail panel. */
export const ExpandButton: FC<ExpandButtonProps> = ({ isExpanded, onToggle }) => {
    const Icon = isExpanded ? ChevronDown : ChevronRight

    return (
        <button
            type="button"
            aria-label={isExpanded ? 'Collapse row' : 'Expand row'}
            aria-expanded={isExpanded}
            // Not a Tab stop and never focused by a click: Space on the table body toggles the row.
            tabIndex={-1}
            onMouseDown={(event) => event.preventDefault()}
            onClick={(event) => {
                // Expanding is not selecting: keep the click off the row.
                event.stopPropagation()
                onToggle()
            }}
            className="flex size-full items-center justify-center text-ink-tertiary hover:text-ink-muted"
        >
            <Icon className={cn('size-3.75', isExpanded && 'text-ink-muted')} />
        </button>
    )
}
