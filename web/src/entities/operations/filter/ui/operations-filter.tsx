import { type FC, useState } from 'react'

import { Search } from 'lucide-react'

import { MultiComboBox } from '@/shared/ui/multi-combo-box'
import { TextInput } from '@/shared/ui/text-input'

import { operationItems, sinkItems, statusItems, tableItems } from '../model/mock'

export const OperationsFilter: FC = () => {
    const [sinks, setSinks] = useState<string[]>(() => sinkItems.map((item) => item.value))
    const [tables, setTables] = useState<string[]>(() => tableItems.map((item) => item.value))
    const [operations, setOperations] = useState<string[]>(() => operationItems.map((item) => item.value))
    const [statuses, setStatuses] = useState<string[]>(() => statusItems.map((item) => item.value))
    const [search, setSearch] = useState('')

    return (
        <>
            <MultiComboBox
                items={sinkItems}
                value={sinks}
                onValueChange={setSinks}
                placeholder="Sinks"
                icon="Sinks"
                className="w-92"
            />
            <MultiComboBox
                items={tableItems}
                value={tables}
                onValueChange={setTables}
                placeholder="Tables"
                icon="Tables"
                className="w-92"
            />
            <MultiComboBox
                items={operationItems}
                value={operations}
                onValueChange={setOperations}
                placeholder="Operations"
                icon="Op"
                className="w-92"
            />
            <MultiComboBox
                items={statusItems}
                value={statuses}
                onValueChange={setStatuses}
                placeholder="Status"
                icon="Status"
                className="w-92"
            />

            <TextInput
                value={search}
                onValueChange={setSearch}
                placeholder="Search table, sink, error…"
                icon={<Search />}
                font="sans"
                className="w-full"
            />
        </>
    )
}
