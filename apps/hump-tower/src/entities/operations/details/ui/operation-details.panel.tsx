import { type FC, useMemo } from 'react'

import { ArrowRight } from 'lucide-react'

import type { Operation } from '@/entities/operations/list/@x/details'
import { OperationBadge } from '@/shared/ui/operation-badge'
import { OperationStatusBadge } from '@/shared/ui/operation-status-badge'
import { formatLatency, formatTime } from '@/shared/utils'

import { toHeaders } from '../model/headers'
import { Details } from './operation-details.details'
import { Headers } from './operation-details.headers'
import { Payload } from './operation-details.payload'
import { SectionCaption } from './operation-details.section-caption'

/** Desktop drawer body: summary card, route, headers and the payload filling the rest. */
export const OperationDetailsPanel: FC<{ operation: Operation }> = ({ operation }) => {
    const headers = useMemo(() => toHeaders(operation.metadata), [operation.metadata])

    return (
        <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto p-4">
            <div className="flex flex-col gap-2.5 rounded-lg border border-hairline bg-surface-2 p-3.5">
                <div className="flex items-center justify-between gap-2">
                    <SectionCaption>Event</SectionCaption>
                    <span className="font-medium font-mono text-2xs text-ink-tertiary">
                        {formatTime(operation.time)}
                    </span>
                </div>
                <div className="flex min-w-0 items-center gap-2 font-medium font-mono text-base text-ink">
                    <OperationBadge op={operation.op} />
                    <span className="truncate">{operation.table}</span>
                    <ArrowRight className="size-3.5 shrink-0 text-ink-tertiary" />
                    <span className="truncate">{operation.sink}</span>
                </div>
                <div className="flex items-center gap-2">
                    <OperationStatusBadge status={operation.status} attempts={operation.attempts} withAttempts />
                    <span className="ml-auto font-mono text-ink-subtle text-sm">
                        {formatLatency(operation.durationMs)}
                    </span>
                </div>
                {operation.error && <p className="wrap-break-word text-danger text-sm">{operation.error}</p>}
            </div>

            <section className="flex flex-col gap-1.5">
                <SectionCaption>Details</SectionCaption>
                <Details operation={operation} />
            </section>

            {headers.length > 0 && (
                <section className="flex flex-col gap-1.5">
                    <SectionCaption>Headers</SectionCaption>
                    <div className="max-h-55 overflow-y-auto">
                        <Headers headers={headers} />
                    </div>
                </section>
            )}

            <Payload
                key={operation.id}
                payload={operation.payload}
                caption={<SectionCaption>Payload</SectionCaption>}
                className="min-h-50"
            />
        </div>
    )
}
