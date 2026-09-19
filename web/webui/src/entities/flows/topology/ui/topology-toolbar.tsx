import { type FC, useState } from 'react'

import { MultiComboBox } from '@/shared/ui/multi-combo-box'

import { sinks, tables } from '../model/mock'

const tableOptions = tables.map((table) => ({ value: table.id, label: table.name }))
const sinkOptions = sinks.map((sink) => ({ value: sink.id, label: sink.name }))

export const TopologyToolbar: FC = () => {
    const [selectedTables, setSelectedTables] = useState(tableOptions.map((option) => option.value))
    const [selectedSinks, setSelectedSinks] = useState(sinkOptions.map((option) => option.value))

    return (
        <div className="flex h-26 shrink-0 items-center justify-between border-hairline border-b bg-canvas px-5 md:h-13">
            <div className="flex w-full flex-col gap-2 md:flex-row md:items-center">
                <MultiComboBox
                    items={tableOptions}
                    value={selectedTables}
                    onValueChange={setSelectedTables}
                    placeholder="Tables"
                    icon="Tables"
                    className="md:w-72"
                />
                <MultiComboBox
                    items={sinkOptions}
                    value={selectedSinks}
                    onValueChange={setSelectedSinks}
                    placeholder="Sinks"
                    icon="Sinks"
                    className="md:w-72"
                />
            </div>
        </div>
    )
}
