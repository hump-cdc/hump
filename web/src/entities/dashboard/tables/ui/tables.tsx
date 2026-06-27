import { Link } from '@tanstack/react-router'

import type { FC } from 'react'

import { ChevronRight, Info, Table2 } from 'lucide-react'

import { Card, CardContent, CardHeader } from '@/shared/ui/card'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/shared/ui/tooltip'
import { cn } from '@/shared/utils'

import { useTables } from '../model/use-tables'

export const Tables: FC = () => {
    const tables = useTables()

    return (
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
                            Tables configured for replication.
                        </TooltipContent>
                    </Tooltip>
                    <span className="text-xs text-ink-tertiary">{tables.length} configured</span>
                </div>
            </CardHeader>

            <CardContent>
                <div className="flex max-h-56 flex-col overflow-y-auto">
                    {tables.length === 0 ? (
                        <span className="px-4 py-3 text-xs text-ink-tertiary">No tables configured</span>
                    ) : (
                        tables.map(({ name, sinks }, index) => (
                            <div
                                key={name}
                                className={cn(
                                    'flex items-center justify-between px-4 py-3',
                                    index < tables.length - 1 && 'border-b border-hairline',
                                )}
                            >
                                <div className="flex items-center gap-2.5">
                                    <Table2 className="size-3.5 text-ink-subtle" />
                                    <span className="font-mono text-sm text-ink">{name}</span>
                                </div>
                                <span className="text-xs font-medium text-ink-subtle">{sinks} sinks</span>
                            </div>
                        ))
                    )}
                </div>
            </CardContent>
        </Card>
    )
}
