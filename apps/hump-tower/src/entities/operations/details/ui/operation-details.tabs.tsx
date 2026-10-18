import { type FC, useMemo } from 'react'

import type { Operation } from '@/entities/operations/list/@x/details'
import { OperationBadge } from '@/shared/ui/operation-badge'
import { OperationStatusBadge } from '@/shared/ui/operation-status-badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/shared/ui/tabs'
import { formatLatency, formatTime } from '@/shared/utils'

import { toHeaders } from '../model/headers'
import { Details } from './operation-details.details'
import { Headers } from './operation-details.headers'
import { Payload } from './operation-details.payload'
import { PropertyRow } from './operation-details.property-row'

const CONTENT_CLASS = 'flex min-h-0 flex-col overflow-y-auto p-4'

/** Text color of the attempts and duration values, as in the list. */
const outcomeTone = ({ status, attempts }: Operation): string =>
    status === 'failed' ? 'text-danger' : attempts > 1 ? 'text-amber' : 'text-ink-muted'

/** Mobile sheet body: Event / Details / Headers / Payload tabs; Headers is hidden without any. */
export const OperationDetailsTabs: FC<{ operation: Operation }> = ({ operation }) => {
    const headers = useMemo(() => toHeaders(operation.metadata), [operation.metadata])

    return (
        <Tabs defaultValue="event" className="min-h-0 flex-1 gap-0">
            <TabsList variant="underline" className="px-2">
                <TabsTrigger value="event">Event</TabsTrigger>
                <TabsTrigger value="details">Details</TabsTrigger>
                {headers.length > 0 && <TabsTrigger value="headers">Headers</TabsTrigger>}
                <TabsTrigger value="payload">Payload</TabsTrigger>
            </TabsList>

            <TabsContent value="event" className={CONTENT_CLASS}>
                <PropertyRow
                    size="comfortable"
                    labelClassName="w-18.5"
                    label="Time"
                    value={formatTime(operation.time)}
                />
                <PropertyRow
                    size="comfortable"
                    labelClassName="w-18.5"
                    label="Operation"
                    value={<OperationBadge op={operation.op} />}
                />
                <PropertyRow size="comfortable" labelClassName="w-18.5" label="Table" value={operation.table} />
                <PropertyRow size="comfortable" labelClassName="w-18.5" label="Sink" value={operation.sink} />
                <PropertyRow
                    size="comfortable"
                    labelClassName="w-18.5"
                    label="Status"
                    value={<OperationStatusBadge status={operation.status} attempts={operation.attempts} />}
                />
                <PropertyRow
                    size="comfortable"
                    labelClassName="w-18.5"
                    label="Attempts"
                    value={`${operation.attempts}×`}
                    valueClassName={outcomeTone(operation)}
                />
                <PropertyRow
                    size="comfortable"
                    labelClassName="w-18.5"
                    label="Duration"
                    value={formatLatency(operation.durationMs)}
                    valueClassName="text-ink-muted"
                />
                {operation.error && <p className="wrap-break-word pt-2.5 text-danger text-sm">{operation.error}</p>}
            </TabsContent>

            <TabsContent value="details" className={CONTENT_CLASS}>
                <Details operation={operation} size="comfortable" />
            </TabsContent>

            {headers.length > 0 && (
                <TabsContent value="headers" className={CONTENT_CLASS}>
                    <Headers headers={headers} />
                </TabsContent>
            )}

            <TabsContent value="payload" className="flex min-h-0 flex-col p-4">
                <Payload payload={operation.payload} className="gap-3" />
            </TabsContent>
        </Tabs>
    )
}
