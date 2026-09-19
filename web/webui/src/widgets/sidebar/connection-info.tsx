import type { FC } from 'react'

import { Book, Database, EthernetPort, Server } from 'lucide-react'

import { useConfig } from '@/entities/config/config'
import { Badge } from '@/shared/ui/badge'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/shared/ui/tooltip'

export const ConnectionInfo: FC = () => {
    const { data: config, isSuccess } = useConfig()
    return (
        <div className="flex max-w-full flex-wrap items-center gap-2.5">
            {isSuccess && (
                <>
                    <Tooltip>
                        <TooltipTrigger>
                            <Badge variant="neutral" border icon={<Server />}>
                                {config.database.host}:{config.database.port}
                            </Badge>
                        </TooltipTrigger>
                        <TooltipContent>Database Host</TooltipContent>
                    </Tooltip>
                    <Tooltip>
                        <TooltipTrigger>
                            <Badge variant="neutral" border title="db" icon={<Database />}>
                                {config.database.name}
                            </Badge>
                        </TooltipTrigger>
                        <TooltipContent>Database Name</TooltipContent>
                    </Tooltip>
                    <Tooltip>
                        <TooltipTrigger>
                            <Badge variant="neutral" border title="slot" icon={<EthernetPort />}>
                                {config.database.slot}
                            </Badge>
                        </TooltipTrigger>
                        <TooltipContent>Database Replication Slot</TooltipContent>
                    </Tooltip>
                    <Tooltip>
                        <TooltipTrigger>
                            <Badge variant="neutral" border title="pub" icon={<Book />}>
                                {config.database.publication}
                            </Badge>
                        </TooltipTrigger>
                        <TooltipContent>Database Publication</TooltipContent>
                    </Tooltip>
                </>
            )}
        </div>
    )
}
