import { type FC, useState } from 'react'

import { sinks, tables } from '@/entities/flows/flows/@x/toolbar'
import { useFlowInspectorStore } from '@/entities/flows/inspector/@x/toolbar'
import { MultiComboBox } from '@/shared/ui/multi-combo-box'
import { Tabs, TabsList, TabsTrigger } from '@/shared/ui/tabs'
import { Toolbar } from '@/shared/ui/toolbar'

import { TOPOLOGY_MODES, type TopologyMode, useTopologyModeStore } from '../model/store'

const tableOptions = tables.map((table) => ({
    value: table.id,
    label: table.name,
}))
const sinkOptions = sinks.map((sink) => ({ value: sink.id, label: sink.name }))

/** Display labels of the topology modes. */
const MODE_LABELS: Record<TopologyMode, string> = {
    summary: 'Summary',
    detailed: 'Detailed',
}

export const FlowsToolbar: FC = () => {
    const [selectedTables, setSelectedTables] = useState(tableOptions.map((option) => option.value))
    const [selectedSinks, setSelectedSinks] = useState(sinkOptions.map((option) => option.value))
    const mode = useTopologyModeStore((state) => state.mode)
    const setMode = useTopologyModeStore((state) => state.setMode)
    const clearSelection = useFlowInspectorStore((state) => state.clear)

    const modeTabs = (
        <Tabs
            value={mode}
            onValueChange={(value) => {
                // Selection ids differ between the views (targets exist only in the detailed one).
                clearSelection()
                setMode(value as TopologyMode)
            }}
            className="ml-auto shrink-0"
        >
            <TabsList className="w-full">
                {TOPOLOGY_MODES.map((option) => (
                    <TabsTrigger key={option} value={option} className="flex-1 py-1 text-xs">
                        {MODE_LABELS[option]}
                    </TabsTrigger>
                ))}
            </TabsList>
        </Tabs>
    )

    return (
        <Toolbar actions={modeTabs}>
            <MultiComboBox
                items={tableOptions}
                value={selectedTables}
                onValueChange={setSelectedTables}
                placeholder="Tables"
                icon="Tables"
                className="w-72"
            />
            <MultiComboBox
                items={sinkOptions}
                value={selectedSinks}
                onValueChange={setSelectedSinks}
                placeholder="Sinks"
                icon="Sinks"
                className="w-72"
            />
        </Toolbar>
    )
}
