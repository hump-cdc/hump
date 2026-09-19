import type { FC, ReactNode } from 'react'

/** Bordered card rendering a small eyebrow label above the selection's identity. */
export const IdentityCard: FC<{ eyebrow: string; children: ReactNode }> = ({ eyebrow, children }) => (
    <div className="flex flex-col gap-2.5 rounded-lg border border-hairline bg-surface-2 p-3.5">
        <span className="font-semibold text-2xs text-ink-tertiary tracking-widest">{eyebrow}</span>
        {children}
    </div>
)
