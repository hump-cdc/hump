import { createHighlighter } from '@tanstack/highlight/core'
import { json } from '@tanstack/highlight/languages/json'
import { plaintext } from '@tanstack/highlight/languages/plaintext'
import { sql } from '@tanstack/highlight/languages/sql'
import { yaml } from '@tanstack/highlight/languages/yaml'

import type { FC } from 'react'
import { useMemo } from 'react'

import { cn } from '@/shared/utils'

import type { CodeGrammar } from './code.grammar'
import { detectGrammar, formatCode } from './code.grammar'

const highlighter = createHighlighter({ languages: [yaml, json, sql, plaintext] })

type CodeProps = {
    /** Source text to display (read-only); its grammar is detected from the content unless `lang` is set. */
    value: string
    /** Grammar to highlight with, skipping detection and formatting (SQL is never detected). */
    lang?: CodeGrammar
    /** Show a line-number gutter. Defaults to `true`. */
    lineNumbers?: boolean
    /** Pretty-print JSON and YAML before highlighting. Defaults to `false` (shown as written). */
    format?: boolean
    /** Wrapper class — control sizing from the consumer (the viewer fills it). */
    className?: string
}

export const Code: FC<CodeProps> = ({ value, lang: grammar, lineNumbers = true, format = false, className }) => {
    const { lang, text } = useMemo(() => {
        if (grammar) return { lang: grammar, text: value }
        return format ? formatCode(value) : { lang: detectGrammar(value), text: value }
    }, [value, grammar, format])
    const html = useMemo(() => highlighter.highlightToHtml(text, { lang, lineNumbers }), [text, lang, lineNumbers])

    return (
        <div
            className={cn('hump-code min-h-0 flex-1 overflow-auto', className)}
            // biome-ignore lint/security/noDangerouslySetInnerHtml: TanStack Highlight escapes `value` while rendering.
            dangerouslySetInnerHTML={{ __html: html }}
        />
    )
}
