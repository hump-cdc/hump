import type { FC, ReactNode } from 'react'

import { Code } from '@/shared/ui/code'
import { CopyButton } from '@/shared/ui/copy-button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/shared/ui/tooltip'

type QueryTooltipProps = {
    query: string
    children: ReactNode
}

/** Shows the query a table is snapshotted with when its trigger is hovered. */
export const QueryTooltip: FC<QueryTooltipProps> = ({ query, children }) => (
    <Tooltip>
        <TooltipTrigger asChild>
            <span className="flex shrink-0">{children}</span>
        </TooltipTrigger>
        <TooltipContent
            side="bottom"
            align="start"
            className="flex w-80 flex-col gap-2 rounded-lg border border-hairline bg-surface-1 p-3 text-ink"
        >
            <div className="flex items-center justify-between gap-2">
                <span className="font-semibold text-ink-subtle text-xs">Snapshot query</span>
                <CopyButton value={query} label="Copy query" />
            </div>
            <Code
                value={query}
                lang="sql"
                lineNumbers={false}
                className="[&_pre]:wrap-break-word rounded-md border border-hairline [--syntax-bg:var(--canvas)] [&_code]:w-auto! [&_pre]:whitespace-pre-wrap"
            />
        </TooltipContent>
    </Tooltip>
)
