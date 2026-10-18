import type { FC } from 'react'

import { type LucideIcon, Rows3, TextAlignJustify } from 'lucide-react'

import { useFlowInspectorStore } from '@/entities/flows/inspector/@x/flows'
import { FlowControlButton } from '@/shared/ui/flow'
import { cn } from '@/shared/utils'

import { TOPOLOGY_MODES, type TopologyMode, useTopologyModeStore } from '../model/mode'

const MODE_BUTTONS: Record<TopologyMode, { label: string; icon: LucideIcon }> = {
    summary: { label: 'Summary', icon: TextAlignJustify },
    detailed: { label: 'Detailed', icon: Rows3 },
}

/** Summary / Detailed toggle appended to the canvas zoom controls. */
export const FlowsModeSwitch: FC = () => {
    const mode = useTopologyModeStore((state) => state.mode)
    const setMode = useTopologyModeStore((state) => state.setMode)
    const clearSelection = useFlowInspectorStore((state) => state.clear)

    return TOPOLOGY_MODES.map((option, index) => {
        const { label, icon: Icon } = MODE_BUTTONS[option]

        return (
            <FlowControlButton
                key={option}
                className={cn(index < TOPOLOGY_MODES.length - 1 && 'flow-controls-joined')}
                title={label}
                aria-label={label}
                aria-pressed={option === mode}
                onClick={() => {
                    if (option === mode) {
                        return
                    }
                    // Selection ids differ between the views (targets exist only in the detailed one).
                    clearSelection()
                    setMode(option)
                }}
            >
                <Icon />
            </FlowControlButton>
        )
    })
}
