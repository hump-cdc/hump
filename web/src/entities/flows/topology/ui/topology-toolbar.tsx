import { type FC, useState } from 'react'

import { MultiComboBox } from '@/shared/ui/multi-combo-box'

import { sinks, tables } from '../model/mock'

const tableOptions = tables.map((table) => ({ value: table.id, label: table.name }))
const sinkOptions = sinks.map((sink) => ({ value: sink.id, label: sink.name }))

export const TopologyToolbar: FC = () => {
    const [selectedTables, setSelectedTables] = useState(tableOptions.map((option) => option.value))
    const [selectedSinks, setSelectedSinks] = useState(sinkOptions.map((option) => option.value))

    return (
        <div className="flex h-26 md:h-13 shrink-0 items-center justify-between border-b border-hairline bg-canvas px-5">
            <div className="flex flex-col md:flex-row md:items-center gap-2 w-full">
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
