import type { FC, ReactNode } from 'react'

import {
    ControlButton,
    Controls,
    type FitViewOptions,
    type ReactFlowState,
    useReactFlow,
    useStore,
    useStoreApi,
} from '@xyflow/react'
import { Maximize, Minus, Plus } from 'lucide-react'

import { FIT_VIEW_PADDING, ZOOM_DURATION } from './flow.consts'
import type { ViewportInsets } from './flow.types'

const selectMaxZoomReached = (state: ReactFlowState): boolean => state.transform[2] >= state.maxZoom
const selectMinZoomReached = (state: ReactFlowState): boolean => state.transform[2] <= state.minZoom

/** Per-side fit-view padding accepted by xyflow's `fitView`. */
type FitViewPadding = NonNullable<FitViewOptions['padding']>

type FlowControlsProps = {
    /** Canvas margins covered by floating overlays; fit view keeps the diagram clear of them. */
    insets: ViewportInsets
    children?: ReactNode
}

/**
 * Zoom-in, zoom-out and fit-view buttons (xyflow's `Controls` layout) that animate the viewport;
 * `children` (extra `FlowControlButton`s) follow in the same card.
 */
export const FlowControls: FC<FlowControlsProps> = ({ insets, children }) => {
    const { zoomIn, zoomOut, fitView } = useReactFlow()
    const store = useStoreApi()
    const maxZoomReached = useStore(selectMaxZoomReached)
    const minZoomReached = useStore(selectMinZoomReached)

    return (
        <Controls showZoom={false} showFitView={false} showInteractive={false}>
            {children}
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
                onClick={() => {
                    const { width, height } = store.getState()
                    void fitView({ duration: ZOOM_DURATION, padding: getFitViewPadding(insets, width, height) })
                }}
            >
                <Maximize />
            </ControlButton>
        </Controls>
    )
}

/**
 * Per-side fit-view padding (px): each inset plus the default {@link FIT_VIEW_PADDING} share of the
 * canvas, which xyflow splits evenly between opposite sides.
 */
const getFitViewPadding = (insets: ViewportInsets, width: number, height: number): FitViewPadding => {
    const x = Math.floor((width - width / (1 + FIT_VIEW_PADDING)) / 2)
    const y = Math.floor((height - height / (1 + FIT_VIEW_PADDING)) / 2)

    return {
        top: `${insets.top + y}px`,
        right: `${insets.right + x}px`,
        bottom: `${insets.bottom + y}px`,
        left: `${insets.left + x}px`,
    }
}
