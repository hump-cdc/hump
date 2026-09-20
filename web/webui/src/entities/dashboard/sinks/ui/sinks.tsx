import { Link } from '@tanstack/react-router'

import type { FC } from 'react'

import { ChevronRight, Info } from 'lucide-react'

import { Card, CardContent, CardHeader } from '@/shared/ui/card'
import { Sink } from '@/shared/ui/sink'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/shared/ui/tooltip'
import { cn } from '@/shared/utils'

import { useSinks } from '../model/use-sinks'

export const Sinks: FC = () => {
    const sinks = useSinks()

    return (
        <Card className="overflow-hidden">
            <CardHeader
                aside={
                    <Link
                        to="/operations"
                        className="flex items-center gap-1.5 font-medium text-ink-subtle text-xs hover:text-ink"
                    >
                        Open operations
                        <ChevronRight className="size-3.5" />
                    </Link>
                }
            >
                <div className="flex items-center gap-2">
                    <span className="font-semibold text-ink text-sm tracking-tight">Sinks</span>
                    <Tooltip>
                        <TooltipTrigger asChild>
                            <span className="inline-flex">
                                <Info className="size-3.5 text-ink-tertiary" />
                            </span>
                        </TooltipTrigger>
                        <TooltipContent className="max-w-56 text-pretty">
                            Configured delivery destinations for replicated rows.
                        </TooltipContent>
                    </Tooltip>
                    <span className="text-ink-tertiary text-xs">{sinks.length} configured</span>
                </div>
            </CardHeader>

            <CardContent>
                <div className="flex max-h-56 flex-col overflow-y-auto">
                    {sinks.length === 0 ? (
                        <span className="px-4 py-3 text-ink-tertiary text-xs">No sinks configured</span>
                    ) : (
                        sinks.map(({ key, type, host }, index) => (
                            <div
                                key={key}
                                className={cn(
                                    'flex items-center justify-between px-4 py-3',
                                    index < sinks.length - 1 && 'border-hairline border-b',
                                )}
                            >
                                <div className="flex items-center gap-2.5">
                                    <span className="size-1.5 rounded-full bg-ink-subtle" />
                                    <Sink type={type} host={host} />
                                </div>
                            </div>
                        ))
                    )}
                </div>
            </CardContent>
        </Card>
    )
}
