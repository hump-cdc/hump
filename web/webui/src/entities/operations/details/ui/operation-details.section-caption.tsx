import type { FC, PropsWithChildren } from 'react'

/** Uppercase caption over a drawer section. */
export const SectionCaption: FC<PropsWithChildren> = ({ children }) => (
    <h3 className="font-semibold text-2xs text-ink-tertiary uppercase tracking-caps">{children}</h3>
)
