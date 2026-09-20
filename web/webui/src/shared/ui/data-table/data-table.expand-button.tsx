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
            onClick={onToggle}
            className="flex size-full items-center justify-center text-ink-tertiary hover:text-ink-muted"
        >
            <Icon className={cn('size-3.75', isExpanded && 'text-ink-muted')} />
        </button>
    )
}
