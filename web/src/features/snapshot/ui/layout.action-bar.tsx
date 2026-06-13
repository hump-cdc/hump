import type { FC, HTMLAttributes, ReactNode } from 'react'

import { cn } from '@/shared/utils'

type ActionBarProps = HTMLAttributes<HTMLDivElement> & {
    asideIcon?: ReactNode
    asideTitle?: ReactNode
    asideSubTitle?: ReactNode
}

/**
 * Bottom action bar pinned to the viewport while the page scrolls.
 * The outer wrapper bleeds past the page padding so its `bg-canvas` backdrop
 * hides content scrolling underneath and adds breathing room below the card.
 */
export const ActionBar: FC<ActionBarProps> = ({
    className,
    asideIcon,
    asideTitle,
    asideSubTitle,
    children,
    ...props
}) => {
    const hasAside = asideIcon || asideTitle || asideSubTitle

    return (
        <>
            <div className="h-0 md:h-0.5"></div>
            <div className="sticky bottom-0 z-10 mt-auto -mx-6 -mb-6 bg-canvas px-6 pt-1 pb-6">
                <div className="pointer-events-none absolute inset-x-0 bottom-full h-8 bg-linear-to-b from-transparent to-canvas" />
                <div
                    className={cn(
                        'flex flex-col gap-3 rounded-xl border border-hairline bg-surface-1 p-3 lg:flex-row lg:items-center',
                        className,
                    )}
                    {...props}
                >
                    {hasAside && (
                        <div className="flex items-center gap-3">
                            {asideIcon && (
                                <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-primary-soft text-primary [&_svg]:size-4.5">
                                    {asideIcon}
                                </div>
                            )}
                            <div className="flex flex-col gap-0.5">
                                {asideTitle && <span className="text-base font-semibold text-ink">{asideTitle}</span>}
                                {asideSubTitle && <span className="text-sm text-ink-subtle">{asideSubTitle}</span>}
                            </div>
                        </div>
                    )}
                    <div className="flex flex-col gap-3 sm:flex-row">{children}</div>
                </div>
            </div>
        </>
    )
}
