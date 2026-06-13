import { Link } from '@tanstack/react-router'

import type { FC } from 'react'

import { Check, ChevronRight, Info, X } from 'lucide-react'

import { Badge } from '@/shared/ui/badge'
import { Card, CardContent, CardHeader } from '@/shared/ui/card'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/shared/ui/tooltip'
import { cn } from '@/shared/utils'

const SINKS = [
    { name: 'kafka1', type: 'kafka', ok: true },
    { name: 'nats-core', type: 'nats', ok: true },
    { name: 'nats-jetstream', type: 'nats', ok: true },
    { name: 'rabbitmq1', type: 'rabbitmq', ok: false },
    { name: 'rabbitmq-stream', type: 'rabbitmq', ok: true },
]

export const Sinks: FC = () => (
    <Card className="overflow-hidden">
        <CardHeader
            aside={
                <Link
                    to="/operations"
                    className="flex items-center gap-1.5 text-xs font-medium text-ink-subtle hover:text-ink"
                >
                    Open operations
                    <ChevronRight className="size-3.5" />
                </Link>
            }
        >
            <div className="flex items-center gap-2">
                <span className="text-sm font-semibold tracking-tight text-ink">Sinks</span>
                <Tooltip>
                    <TooltipTrigger asChild>
                        <span className="inline-flex">
                            <Info className="size-3.5 text-ink-tertiary" />
                        </span>
                    </TooltipTrigger>
                    <TooltipContent className="max-w-56 text-pretty">
                        Destinations Hump delivers replicated rows to.
                    </TooltipContent>
                </Tooltip>
                <span className="text-xs text-ink-tertiary">{SINKS.length} presets</span>
            </div>
        </CardHeader>

        <CardContent>
            <div className="flex flex-col">
                {SINKS.map(({ name, type, ok }, index) => (
                    <div
                        key={name}
                        className={cn(
                            'flex items-center justify-between px-4 py-3',
                            index < SINKS.length - 1 && 'border-b border-hairline',
                        )}
                    >
                        <div className="flex items-center gap-2.5">
                            <span className={cn('size-1.5 rounded-full', ok ? 'bg-success' : 'bg-danger')} />
                            <span className="text-sm font-medium text-ink">{name}</span>
                            <Badge variant="neutral">{type}</Badge>
                        </div>
                        {ok ? <Check className="size-3 text-success" /> : <X className="size-3 text-danger" />}
                    </div>
                ))}
            </div>
        </CardContent>
    </Card>
)
