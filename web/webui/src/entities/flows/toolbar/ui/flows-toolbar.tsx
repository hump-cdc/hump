import type { FC } from 'react'

import { Table2, Workflow } from 'lucide-react'

import { type FlowsFilterSelection, useFlowsFilterStore, useFlowsTopology } from '@/entities/flows/flows/@x/toolbar'
import { MultiComboBox } from '@/shared/ui/multi-combo-box'
import { Toolbar } from '@/shared/ui/toolbar'

export const FlowsToolbar: FC = () => {
    const { tables, detailedSinks: sinks } = useFlowsTopology()
    const { tables: selectedTables, sinks: selectedSinks, setTables, setSinks } = useFlowsFilterStore()

    const tableOptions = tables.map((table) => ({ value: table.id, label: table.name }))
    const sinkOptions = sinks.map((sink) => ({ value: sink.id, label: `${sink.name} · ${sink.host}` }))

    return (
        <Toolbar>
            <MultiComboBox
                clearToAll
                items={tableOptions}
                {...bindSelection(tableOptions, selectedTables, setTables)}
                placeholder="Tables"
                hint="Tables to show on the flow, with their connections"
                icon={<Table2 />}
                className="w-1/2"
            />
            <MultiComboBox
                clearToAll
                items={sinkOptions}
                {...bindSelection(sinkOptions, selectedSinks, setSinks)}
                placeholder="Sinks"
                hint="Sink endpoints to show on the flow, with their connections"
                icon={<Workflow />}
                className="w-1/2"
            />
        </Toolbar>
    )
}

/** Maps a stored selection to `value`/`onValueChange` props, storing `all` once every option is picked. */
const bindSelection = (
    options: ReadonlyArray<{ value: string }>,
    selection: FlowsFilterSelection,
    onChange: (value: FlowsFilterSelection) => void,
) => ({
    value: selection === 'all' ? options.map((option) => option.value) : selection,
    onValueChange: (value: string[]) => onChange(value.length === options.length ? 'all' : value),
})
