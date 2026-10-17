import { Link } from '@tanstack/react-router'

import type { FC } from 'react'

import { History, ListChecks, RefreshCw, TriangleAlert } from 'lucide-react'

import { Button } from '@/shared/ui/button'
import { useSwitchViewValue } from '@/shared/utils'

import { useStopSnapshot } from '../../model/snapshot'
import type { SnapshotState } from '../../model/types'
import { ActionBar } from '../action-bar'

export const DoneActionBar: FC = () => {
    const data = useSwitchViewValue<SnapshotState>()
    const { mutate: clear, isPending } = useStopSnapshot()

    if (data?.state !== 'done') return null

    const hasFailed = data.tables.some((table) => table.status === 'error')

    return (
        <ActionBar
            className="lg:justify-between"
            asideIcon={hasFailed ? <TriangleAlert className="text-danger" /> : <History />}
            asideIconClassName={hasFailed ? 'bg-danger-soft' : undefined}
            asideTitle={hasFailed ? 'Snapshot is incomplete' : 'Snapshot is done'}
            asideSubTitle={
                hasFailed
                    ? "Some tables didn't load. Run again to retry."
                    : 'Done. Sinks are seeded with requested data.'
            }
        >
            <Button variant="outline" className="w-full lg:w-auto" asChild>
                <Link to="/operations">
                    <ListChecks />
                    View operations
                </Link>
            </Button>
            <Button onClick={() => clear()} disabled={isPending} className="w-full lg:w-auto">
                <RefreshCw />
                Run another snapshot
            </Button>
        </ActionBar>
    )
}
