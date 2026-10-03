import type { FC } from 'react'

import { CornerUpLeft } from 'lucide-react'

import { Tooltip, TooltipContent, TooltipTrigger } from '@/shared/ui/tooltip'

type RestoreSuggestionProps = {
    suggestion: string
    onRestore: () => void
}

export const RestoreSuggestion: FC<RestoreSuggestionProps> = ({ suggestion, onRestore }) => (
    <Tooltip>
        <TooltipTrigger asChild>
            <button
                type="button"
                tabIndex={-1}
                aria-label="Restore suggested filter"
                onClick={onRestore}
                className="flex size-5 items-center justify-center rounded text-ink-subtle outline-none transition-colors hover:text-ink focus-visible:ring-2 focus-visible:ring-primary/40"
            >
                <CornerUpLeft className="size-3.5" />
            </button>
        </TooltipTrigger>
        <TooltipContent className="max-w-80 font-mono">{suggestion}</TooltipContent>
    </Tooltip>
)
