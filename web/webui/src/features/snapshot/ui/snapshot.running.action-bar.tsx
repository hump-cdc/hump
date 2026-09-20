import { Link } from '@tanstack/react-router'

import type { FC } from 'react'

import { GitBranch, ListChecks, Square } from 'lucide-react'

import { Button } from '@/shared/ui/button'

import { useSnapshot } from '../model/use-snapshot'
import { ActionBar } from './layout.action-bar'

export const RunningActionBar: FC = () => {
    const { stop } = useSnapshot()

    return (
        <ActionBar className="lg:justify-end">
            <Button variant="outline" className="w-full lg:w-auto" asChild>
                <Link to="/">
                    <GitBranch />
                    View flows
                </Link>
            </Button>
            <Button variant="outline" className="w-full lg:w-auto" asChild>
                <Link to="/operations">
                    <ListChecks />
                    View operations
                </Link>
            </Button>
            <Button variant="secondary" onClick={stop} className="w-full lg:w-auto">
                <Square className="text-danger" />
                Stop snapshot
            </Button>
        </ActionBar>
    )
}
