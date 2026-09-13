import type { FC } from 'react'

import { OperationsFilter } from '@/entities/operations/filter'
import { OperationsList } from '@/entities/operations/list'
import { Toolbar } from '@/shared/ui/toolbar'

export const Operations: FC = () => (
    <div className="flex flex-1 flex-col gap-4 p-6">
        <Toolbar>
            <OperationsFilter />
        </Toolbar>
        <OperationsList />
    </div>
)
