import { Link, type LinkProps } from '@tanstack/react-router'

import type { FC } from 'react'

import { ArrowUpRight, ChartColumn, List, type LucideIcon, Settings2, Workflow } from 'lucide-react'

const LINKS: { to: LinkProps['to']; icon: LucideIcon; title: string; subtitle: string }[] = [
    { to: '/configuration', icon: Settings2, title: 'Configuration', subtitle: 'View & edit config' },
    { to: '/operations', icon: List, title: 'Operations', subtitle: 'Live delivery stream' },
    { to: '/flows', icon: Workflow, title: 'Flows', subtitle: 'Table → sink topology' },
    { to: '/stats', icon: ChartColumn, title: 'Stats', subtitle: 'Metrics & charts' },
]

export const QuickLinks: FC = () => (
    <>
        {LINKS.map(({ to, icon: Icon, title, subtitle }) => (
            <Link
                key={to}
                to={to}
                className="flex items-center gap-3 rounded-lg border border-hairline bg-surface-1 px-4 py-3.5 transition-colors"
            >
                <span className="flex size-8 items-center justify-center rounded-lg bg-primary-soft">
                    <Icon className="size-4 text-primary" />
                </span>
                <div className="flex flex-1 flex-col gap-0.5">
                    <div className="flex items-center gap-1.5">
                        <span className="text-sm font-semibold tracking-tight text-ink">{title}</span>
                    </div>
                    <span className="text-xs text-ink-subtle">{subtitle}</span>
                </div>
                <ArrowUpRight className="size-4 text-ink-tertiary" />
            </Link>
        ))}
    </>
)
