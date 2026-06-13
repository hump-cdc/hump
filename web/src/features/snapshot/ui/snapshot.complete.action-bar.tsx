import { Link } from '@tanstack/react-router'

import type { FC } from 'react'

import { History, ListChecks, RefreshCw } from 'lucide-react'

import { Button } from '@/shared/ui/button'

import { useSnapshot } from '../model/use-snapshot'
import { ActionBar } from './layout.action-bar'

export const CompleteActionBar: FC = () => {
    const { reset } = useSnapshot()

    return (
        <ActionBar
            className="lg:justify-between"
            asideIcon={<History />}
            asideTitle="Snapshot is done"
            asideSubTitle="Re-run anytime from the Snapshot page"
        >
            <Button variant="outline" className="w-full lg:w-auto" asChild>
                <Link to="/operations">
                    <ListChecks />
                    View operations
                </Link>
            </Button>
            <Button onClick={reset} className="w-full lg:w-auto">
                <RefreshCw />
                Run another snapshot
            </Button>
        </ActionBar>
    )
}
