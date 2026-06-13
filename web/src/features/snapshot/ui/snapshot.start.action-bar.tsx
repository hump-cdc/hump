import type { FC } from 'react'

import { Camera } from 'lucide-react'

import { Button } from '@/shared/ui/button'
import { formatNumberCompact } from '@/shared/utils'

import { estimatedRows, selectedTables } from '../model/selectors'
import { useSnapshot } from '../model/use-snapshot'
import { ActionBar } from './layout.action-bar'

export const StartActionBar: FC = () => {
    const { selectedIds, run } = useSnapshot()

    const selectedCount = selectedIds.size
    const chosen = selectedTables(selectedIds)
    const totalEstimated = estimatedRows(chosen)

    return (
        <ActionBar
            className="lg:justify-between"
            asideIcon={<Camera />}
            asideTitle={
                <>
                    {selectedCount} {selectedCount === 1 ? 'table' : 'tables'} selected · ≈{' '}
                    {formatNumberCompact(totalEstimated)} rows
                </>
            }
            asideSubTitle="Every row will be re-streamed to all connected sinks"
        >
            <Button onClick={run} disabled={selectedCount === 0} className="w-full lg:w-auto">
                <Camera />
                Run snapshot
            </Button>
        </ActionBar>
    )
}
