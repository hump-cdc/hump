import type { FC } from 'react'

import { ControlButton, Controls, type ReactFlowState, useReactFlow, useStore } from '@xyflow/react'
import { Maximize, Minus, Plus } from 'lucide-react'

import { ZOOM_DURATION } from './flow.consts'

const selectMaxZoomReached = (state: ReactFlowState): boolean => state.transform[2] >= state.maxZoom
const selectMinZoomReached = (state: ReactFlowState): boolean => state.transform[2] <= state.minZoom

/** Zoom-in, zoom-out and fit-view buttons (xyflow's `Controls` layout) that animate the viewport. */
export const FlowControls: FC = () => {
    const { zoomIn, zoomOut, fitView } = useReactFlow()
    const maxZoomReached = useStore(selectMaxZoomReached)
    const minZoomReached = useStore(selectMinZoomReached)

    return (
        <Controls showZoom={false} showFitView={false} showInteractive={false}>
            <ControlButton
                className="react-flow__controls-zoomin"
                title="Zoom In"
                aria-label="Zoom In"
                disabled={maxZoomReached}
                onClick={() => zoomIn({ duration: ZOOM_DURATION })}
            >
                <Plus />
            </ControlButton>
            <ControlButton
                className="react-flow__controls-zoomout"
                title="Zoom Out"
                aria-label="Zoom Out"
                disabled={minZoomReached}
                onClick={() => zoomOut({ duration: ZOOM_DURATION })}
            >
                <Minus />
            </ControlButton>
            <ControlButton
                className="react-flow__controls-fitview"
                title="Fit View"
                aria-label="Fit View"
                onClick={() => fitView({ duration: ZOOM_DURATION })}
            >
                <Maximize />
            </ControlButton>
        </Controls>
    )
}
