import type { FC } from 'react'

import { Database, GitCommitVertical, Server } from 'lucide-react'

import { Badge } from '@/shared/ui/badge'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/shared/ui/tooltip'

export const ConnectionInfo: FC = () => (
    <div className="flex flex-wrap items-center gap-2.5 max-w-full">
        <Tooltip>
            <TooltipTrigger>
                <Badge variant="neutral" border icon={<Server />}>
                    localhost:5432
                </Badge>
            </TooltipTrigger>
            <TooltipContent>Postgres host</TooltipContent>
        </Tooltip>
        <Tooltip>
            <TooltipTrigger>
                <Badge variant="neutral" border title="db" icon={<Database />}>
                    users
                </Badge>
            </TooltipTrigger>
            <TooltipContent>Database name</TooltipContent>
        </Tooltip>
        <Tooltip>
            <TooltipTrigger>
                <Badge variant="neutral" border title="slot" icon={<GitCommitVertical />}>
                    hump_repl
                </Badge>
            </TooltipTrigger>
            <TooltipContent>Postgres replication slot</TooltipContent>
        </Tooltip>
    </div>
)
