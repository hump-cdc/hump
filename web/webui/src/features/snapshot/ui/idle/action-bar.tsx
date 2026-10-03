import type { FC } from 'react'
import { Fragment } from 'react'

import { Camera, TriangleAlert } from 'lucide-react'

import { Button } from '@/shared/ui/button'
import { Mark } from '@/shared/ui/mark'
import { NumberFlow } from '@/shared/ui/number-flow'

import { useStartSnapshot } from '../../model/snapshot'
import { useSelectedTables, useTables, useTablesEstimate } from '../../model/tables'
import { ActionBar } from '../action-bar'

export const IdleActionBar: FC = () => {
    const selectedTables = useSelectedTables()
    const wheres = useTables((store) => store.wheres)
    const estimates = useTablesEstimate()

    const failed = selectedTables.filter((table) => estimates[table.id]?.status === 'error')

    const isEstimating = selectedTables.some(
        (table) => estimates[table.id]?.status !== 'ready' && estimates[table.id]?.status !== 'error',
    )
    const totalEstimated = selectedTables.reduce((total, table) => {
        const estimate = estimates[table.id]
        return estimate?.status === 'ready' ? total + estimate.count : total
    }, 0)
    const hasErrors = failed.length > 0

    const { mutate: run, isPending } = useStartSnapshot()

    return (
        <ActionBar
            className="lg:justify-between"
            asideIcon={hasErrors ? <TriangleAlert className="text-danger" /> : <Camera />}
            asideIconClassName={hasErrors ? 'bg-danger-soft' : undefined}
            asideTitle={
                <>
                    <NumberFlow value={selectedTables.length} />
                    {selectedTables.length === 1 ? ' table' : ' tables'} selected (≈
                    <NumberFlow value={totalEstimated} kind="compact" /> rows)
                    {hasErrors && (
                        <>
                            {' · '}
                            <NumberFlow value={failed.length} />
                            {failed.length === 1 ? ' filter error' : ' filter errors'}
                        </>
                    )}
                </>
            }
            asideSubTitle={
                hasErrors ? (
                    <span className="text-danger">
                        Fix the <Mark className="text-danger">WHERE</Mark> {failed.length === 1 ? 'error' : 'errors'} in{' '}
                        {failed.map((table, index) => (
                            <Fragment key={table.id}>
                                {index > 0 && ', '}
                                <Mark className="text-danger">{table.name}</Mark>
                            </Fragment>
                        ))}{' '}
                        to run the snapshot
                    </span>
                ) : (
                    'Every row will be re-streamed to the connected sinks'
                )
            }
        >
            <Button
                onClick={() =>
                    run(
                        selectedTables.map((table) => ({
                            schema: table.schema,
                            table: table.table,
                            where: wheres[table.id] ?? table.suggestedWhere,
                        })),
                    )
                }
                disabled={selectedTables.length === 0 || hasErrors || isPending || isEstimating}
                className="w-full lg:w-auto"
            >
                <Camera />
                Run snapshot
            </Button>
        </ActionBar>
    )
}
