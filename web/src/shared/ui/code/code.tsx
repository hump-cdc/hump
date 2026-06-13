import type { FC } from 'react'
import { useEffect, useState } from 'react'

import yaml from '@shikijs/langs/yaml'
import { createHighlighterCore, type HighlighterCore } from 'shiki/core'
import { createJavaScriptRegexEngine } from 'shiki/engine/javascript'

import { cn } from '@/shared/utils'

import { humpCode } from './code.theme'

let highlighterPromise: Promise<HighlighterCore> | undefined

const getHighlighter = (): Promise<HighlighterCore> => {
    highlighterPromise ??= createHighlighterCore({
        themes: [humpCode],
        langs: [yaml],
        engine: createJavaScriptRegexEngine(),
    })
    return highlighterPromise
}

type CodeProps = {
    /** Source text to display (read-only). */
    value: string
    /** Wrapper class — control sizing from the consumer (the viewer fills it). */
    className?: string
}

export const Code: FC<CodeProps> = ({ value, className }) => {
    const [html, setHtml] = useState<string | null>(null)

    useEffect(() => {
        let active = true
        getHighlighter().then((highlighter) => {
            if (!active) return
            setHtml(
                highlighter.codeToHtml(value, {
                    lang: 'yaml',
                    theme: 'hump-code',
                }),
            )
        })
        return () => {
            active = false
        }
    }, [value])

    return (
        <div className={cn('hump-code min-h-0 flex-1 overflow-auto', className)}>
            {html ? (
                // biome-ignore lint/security/noDangerouslySetInnerHtml: Shiki output is server-grade sanitized HTML built from `value`.
                <div dangerouslySetInnerHTML={{ __html: html }} />
            ) : (
                <pre className="hump-code-fallback">{value}</pre>
            )}
        </div>
    )
}
