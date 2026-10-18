import { type FC, useState } from 'react'

import { ArrowRight, EqualApproximately, Info, Loader2, Table2, TriangleAlert } from 'lucide-react'

import { Mark } from '@/shared/ui/mark'
import { Sink } from '@/shared/ui/sink'
import type { TableRowColumn } from '@/shared/ui/table'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/shared/ui/tooltip'
import { WhereInput } from '@/shared/ui/where-input'
import { cn, formatNumberCompact } from '@/shared/utils'

import { snapshotQuery } from '../../model/selectors'
import type { SnapshotTable, TableEstimate } from '../../model/types'
import { QueryTooltip } from '../query-tooltip'

type TableRowData = {
    table: SnapshotTable
    selected: boolean
    where: string
    estimate: TableEstimate | undefined
    onWhereCommit: (where: string) => void
}

export const TABLE_COLUMNS: TableRowColumn<TableRowData>[] = [
    {
        key: 'name',
        header: 'Table',
        className: 'lg:w-56 xl:w-64',
        cell: (d) => {
            const query = snapshotQuery(d.table, d.where)

            return (
                <div className="flex min-w-0 items-center justify-between gap-2 lg:w-56 xl:w-64">
                    <span className="flex min-w-0 items-center gap-2">
                        {d.selected ? (
                            <QueryTooltip query={query}>
                                <Table2 className="size-4 text-primary" />
                            </QueryTooltip>
                        ) : (
                            <Table2 className="size-4 shrink-0 text-ink-subtle" />
                        )}
                        <span className="truncate font-mono text-base text-ink">{d.table.name}</span>
                    </span>
                    <span className="flex shrink-0 items-center lg:hidden">
                        <EstimateRowCount estimate={d.estimate} />
                    </span>
                </div>
            )
        },
    },
    {
        key: 'filter',
        header: 'Filter',
        className: 'flex-1',
        cell: (d) => <FilterCell {...d} />,
    },
    {
        key: 'rows',
        header: 'Est. rows',
        className: 'lg:w-24',
        cell: (d) => (
            <div className="hidden lg:flex lg:w-24">
                <EstimateRowCount estimate={d.estimate} />
            </div>
        ),
    },
    {
        key: 'sinks',
        header: 'Sinks',
        className: 'lg:w-80',
        cell: (d) => (
            <div className="flex flex-col gap-1.5 overflow-hidden lg:w-80">
                {d.table.sinks.map((sink) => (
                    <div key={`${sink.type} ${sink.host} ${sink.target}`} className="flex min-w-0 items-center gap-1.5">
                        <Sink type={sink.type} host={sink.host} className="shrink-0" />
                        <ArrowRight className="size-3 shrink-0 text-ink-tertiary" />
                        <Mark className="truncate font-mono text-ink-subtle text-xs">{sink.target}</Mark>
                    </div>
                ))}
            </div>
        ),
    },
]

const EstimateRowCount: FC<{ estimate: TableEstimate | undefined }> = ({ estimate }) => {
    if (estimate?.status === 'loading') return <Loader2 className="size-3.5 animate-spin text-ink-subtle" />
    if (estimate?.status !== 'ready') return null

    return (
        <span
            className={cn('flex items-center gap-1 font-mono text-base text-ink', estimate.fetching && 'animate-pulse')}
        >
            {formatNumberCompact(estimate.count)}
            {estimate.rough && (
                <Tooltip>
                    <TooltipTrigger asChild>
                        <Mark
                            aria-label="Estimated from table statistics"
                            className="flex shrink-0 items-center p-0.5 text-ink-subtle"
                        >
                            <EqualApproximately className="size-3.5" />
                        </Mark>
                    </TooltipTrigger>
                    <TooltipContent
                        side="bottom"
                        className="w-64 rounded-lg border border-hairline bg-surface-1 p-3 font-normal text-ink text-sm"
                    >
                        Estimated from table statistics. The actual row count may differ.
                    </TooltipContent>
                </Tooltip>
            )}
        </span>
    )
}

/** Row filter editor; the edited text stays local until the input loses focus. */
const FilterCell: FC<TableRowData> = ({ table, selected, where, estimate, onWhereCommit }) => {
    const [draft, setDraft] = useState(where)
    const [committed, setCommitted] = useState(where)

    if (committed !== where) {
        setCommitted(where)
        setDraft(where)
    }

    const error = estimate?.status === 'error' ? estimate.error : undefined

    return (
        <div className="flex min-w-0 flex-1 flex-col gap-1.5">
            <div className="flex items-center gap-2">
                <WhereInput
                    value={draft}
                    onValueChange={setDraft}
                    onBlur={() => onWhereCommit(draft)}
                    suggestion={table.suggestedWhere}
                    fields={table.columns}
                    disabled={!selected}
                    invalid={Boolean(error)}
                />
                <Tooltip>
                    <TooltipTrigger asChild>
                        <button
                            type="button"
                            aria-label="About row filter"
                            className="flex size-6 shrink-0 items-center justify-center rounded-full text-ink-subtle transition-colors hover:text-ink"
                        >
                            <Info className="size-4" />
                        </button>
                    </TooltipTrigger>
                    <TooltipContent
                        side="bottom"
                        className="flex w-80 flex-col gap-2 rounded-lg border border-hairline bg-surface-1 p-3 font-normal text-ink text-sm"
                    >
                        <p>
                            Optional SQL condition for this table. Only rows that match it are re-read and re-emitted;
                            leave empty to snapshot the full table.
                        </p>
                        <p>Row estimate is recalculated once you change the condition.</p>
                        <p className="rounded-md border border-hairline bg-canvas px-2.5 py-1.5 font-mono text-xs">
                            <span className="text-ink-subtle">e.g.</span> status = 'active' AND id &gt; 1000
                        </p>
                    </TooltipContent>
                </Tooltip>
            </div>
            {error && (
                <span className="flex items-start gap-1.5 font-mono text-danger text-xs">
                    <TriangleAlert className="mt-px size-3.5 shrink-0" />
                    <span className="wrap-break-word min-w-0">{error}</span>
                </span>
            )}
        </div>
    )
}
