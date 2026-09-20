import { createHighlighter } from '@tanstack/highlight/core'
import { yaml } from '@tanstack/highlight/languages/yaml'

import type { FC } from 'react'
import { useMemo } from 'react'

import { cn } from '@/shared/utils'

const highlighter = createHighlighter({ languages: [yaml] })

type CodeProps = {
    /** Source text to display (read-only). */
    value: string
    /** Wrapper class — control sizing from the consumer (the viewer fills it). */
    className?: string
}

export const Code: FC<CodeProps> = ({ value, className }) => {
    const html = useMemo(() => highlighter.highlightToHtml(value, { lang: 'yaml', lineNumbers: true }), [value])

    return (
        <div
            className={cn('hump-code min-h-0 flex-1 overflow-auto', className)}
            // biome-ignore lint/security/noDangerouslySetInnerHtml: TanStack Highlight escapes `value` while rendering.
            dangerouslySetInnerHTML={{ __html: html }}
        />
    )
}
