import { Link } from '@tanstack/react-router'

import type { FC } from 'react'

import { ChevronRight, Info, Table2 } from 'lucide-react'

import { Card, CardContent, CardHeader } from '@/shared/ui/card'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/shared/ui/tooltip'
import { cn } from '@/shared/utils'

const TABLES = [
    { name: 'public.users', sinks: 5 },
    { name: 'public.posts', sinks: 3 },
    { name: 'public.comments', sinks: 4 },
    { name: 'public.events', sinks: 2 },
    { name: 'public.sessions', sinks: 1 },
]

export const Tables: FC = () => (
    <Card className="overflow-hidden">
        <CardHeader
            aside={
                <Link
                    to="/flows"
                    className="flex items-center gap-1.5 text-xs font-medium text-ink-subtle hover:text-ink"
                >
                    Open flows
                    <ChevronRight className="size-3.5" />
                </Link>
            }
        >
            <div className="flex items-center gap-2">
                <span className="text-sm font-semibold tracking-tight text-ink">Tables</span>
                <Tooltip>
                    <TooltipTrigger asChild>
                        <span className="inline-flex">
                            <Info className="size-3.5 text-ink-tertiary" />
                        </span>
                    </TooltipTrigger>
                    <TooltipContent className="max-w-56 text-pretty">
                        Postgres tables configured for replication.
                    </TooltipContent>
                </Tooltip>
                <span className="text-xs text-ink-tertiary">{TABLES.length} configured</span>
            </div>
        </CardHeader>

        <CardContent>
            <div className="flex flex-col">
                {TABLES.map(({ name, sinks }, index) => (
                    <div
                        key={name}
                        className={cn(
                            'flex items-center justify-between px-4 py-3',
                            index < TABLES.length - 1 && 'border-b border-hairline',
                        )}
                    >
                        <div className="flex items-center gap-2.5">
                            <Table2 className="size-3.5 text-ink-subtle" />
                            <span className="font-mono text-sm text-ink">{name}</span>
                        </div>
                        <span className="text-xs font-medium text-ink-subtle">{sinks} sinks</span>
                    </div>
                ))}
            </div>
        </CardContent>
    </Card>
)
