import type { FC } from 'react'

type MatchedLabelProps = {
    label: string
    matches: number[]
}

/** Label with the characters matched by the typed prefix emphasised. */
export const MatchedLabel: FC<MatchedLabelProps> = ({ label, matches }) => {
    const parts = []
    let pos = 0
    for (let i = 0; i < matches.length; i += 2) {
        const [start, end] = [matches[i], matches[i + 1]]
        if (start > pos) parts.push(<span key={`t${pos}`}>{label.slice(pos, start)}</span>)
        parts.push(
            <span key={`m${start}`} className="text-primary">
                {label.slice(start, end)}
            </span>,
        )
        pos = end
    }
    if (pos < label.length) parts.push(<span key={`t${pos}`}>{label.slice(pos)}</span>)
    return <span className="min-w-0 flex-1 truncate">{parts}</span>
}
