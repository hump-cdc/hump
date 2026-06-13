import type { FC } from 'react'
import { useCallback } from 'react'

import { Check, Copy, FileCode2 } from 'lucide-react'

import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Code } from '@/shared/ui/code'
import { useClipboard } from '@/shared/utils'

import { CONFIG_FILENAME, MOCK_CONFIG_YAML } from '../model/mock'

export const SourceViewer: FC = () => {
    const { copied, copy } = useClipboard()
    const handleCopy = useCallback(() => {
        copy(MOCK_CONFIG_YAML)
    }, [copy])

    return (
        <div className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-lg border border-hairline bg-surface-1">
            <div className="flex items-center justify-between border-b border-hairline px-3 py-2">
                <div className="flex items-center gap-2.5">
                    <FileCode2 className="size-4 text-ink-subtle" />
                    <span className="font-mono text-sm text-ink">{CONFIG_FILENAME}</span>
                    <Badge variant="neutral">read-only</Badge>
                </div>
                <Button variant="ghost" className="size-8.5 px-0" aria-label="Copy configuration" onClick={handleCopy}>
                    {copied ? <Check className="text-success" /> : <Copy />}
                </Button>
            </div>

            <Code value={MOCK_CONFIG_YAML} />
        </div>
    )
}
