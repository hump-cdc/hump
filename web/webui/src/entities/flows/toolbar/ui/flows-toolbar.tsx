import { type FC, useState } from 'react'

import { useFlowsTopology } from '@/entities/flows/flows/@x/toolbar'
import { useFlowInspectorStore } from '@/entities/flows/inspector/@x/toolbar'
import { MultiComboBox } from '@/shared/ui/multi-combo-box'
import { Tabs, TabsList, TabsTrigger } from '@/shared/ui/tabs'
import { Toolbar } from '@/shared/ui/toolbar'

import { TOPOLOGY_MODES, type TopologyMode, useTopologyModeStore } from '../model/store'

/** Display labels of the topology modes. */
const MODE_LABELS: Record<TopologyMode, string> = {
    summary: 'Summary',
    detailed: 'Detailed',
}

/** A filter option; its `value` is the node id. */
type Option = { value: string; label: string }

/**
 * Selection kept as the options switched off, so every table or sink starts selected, including
 * those arriving after mount with the first metrics response or appearing later.
 */
const useExclusion = (options: Option[]) => {
    const [excluded, setExcluded] = useState<ReadonlySet<string>>(() => new Set())

    return {
        value: options.filter((option) => !excluded.has(option.value)).map((option) => option.value),
        onValueChange: (value: string[]) => {
            const selected = new Set(value)
            setExcluded(new Set(options.filter((option) => !selected.has(option.value)).map((option) => option.value)))
        },
    }
}

export const FlowsToolbar: FC = () => {
    const { tables, detailedSinks } = useFlowsTopology()
    const tableOptions = tables.map((table) => ({ value: table.id, label: table.name }))
    // Sinks of one type differ by host only, so the label carries both.
    const sinkOptions = detailedSinks.map((sink) => ({ value: sink.id, label: `${sink.name} · ${sink.host}` }))
    const tableFilter = useExclusion(tableOptions)
    const sinkFilter = useExclusion(sinkOptions)
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
            <MultiComboBox items={tableOptions} {...tableFilter} placeholder="Tables" icon="Tables" className="w-72" />
            <MultiComboBox items={sinkOptions} {...sinkFilter} placeholder="Sinks" icon="Sinks" className="w-72" />
        </Toolbar>
    )
}
