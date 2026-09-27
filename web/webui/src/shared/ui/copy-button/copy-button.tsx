import type { FC } from 'react'

import { Check, Copy } from 'lucide-react'

import { cn, useClipboard } from '@/shared/utils'

type CopyButtonProps = {
    value: string
    label: string
    className?: string
}

export const CopyButton: FC<CopyButtonProps> = ({ value, label, className }) => {
    const { copied, copy } = useClipboard()

    return (
        <button
            type="button"
            aria-label={label}
            onClick={() => copy(value)}
            className={cn(
                'flex shrink-0 cursor-pointer items-center justify-center text-ink-tertiary transition-colors hover:text-ink',
                className,
            )}
        >
            {copied ? <Check className="size-3.25 text-success" /> : <Copy className="size-3.25" />}
        </button>
    )
}
