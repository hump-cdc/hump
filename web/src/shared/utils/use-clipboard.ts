import { useCallback, useState } from 'react'

export interface UseClipboard {
    copied: boolean
    copy: (text: string) => void
}

export const useClipboard = (resetDelay: number = 1500): UseClipboard => {
    const [copied, setCopied] = useState(false)

    const copy = useCallback(
        (text: string) => {
            navigator.clipboard
                .writeText(text)
                .then(() => {
                    setCopied(true)
                    window.setTimeout(() => setCopied(false), resetDelay)
                })
                .catch(() => {})
        },
        [resetDelay],
    )

    return { copied, copy }
}
