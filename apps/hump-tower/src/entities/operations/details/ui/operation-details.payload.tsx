import type { FC, ReactNode } from 'react'

import { Code } from '@/shared/ui/code'
import { CopyButton } from '@/shared/ui/copy-button'
import { NoPayloadPlaceholder } from '@/shared/ui/placeholder'
import { cn } from '@/shared/utils'

type PayloadProps = {
    payload: string
    /** Caption rendered above the body, if any. */
    caption?: ReactNode
    className?: string
}

/** Message body as-is with a copy button; an empty one shows a placeholder. */
export const Payload: FC<PayloadProps> = ({ payload, caption, className }) => (
    <div className={cn('flex min-h-0 flex-1 flex-col gap-2', className)}>
        {caption && <div className="flex items-center justify-between gap-2">{caption}</div>}
        {payload ? (
            <div className="relative flex min-h-0 flex-1 flex-col overflow-hidden rounded-lg border border-hairline bg-canvas">
                <Code value={payload} format lineNumbers={false} className="[--syntax-bg:var(--canvas)]" />
                <CopyButton
                    value={payload}
                    label="Copy payload"
                    className="absolute top-3 right-4 size-6.5 rounded-md border border-hairline bg-surface-2"
                />
            </div>
        ) : (
            <NoPayloadPlaceholder />
        )}
    </div>
)
