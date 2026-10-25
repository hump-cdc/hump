import type { FC } from 'react'

import { afterAll, afterEach, beforeAll, describe, expect, mock, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { act, cleanup, fireEvent, render } from '@testing-library/react'
import { ReactFlowProvider, type ReactFlowState, useStoreApi } from '@xyflow/react'

import { ZOOM_DURATION } from './flow.consts'
import { FlowControls } from './flow.controls'

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(cleanup)

type Store = ReturnType<typeof useStoreApi>

/** xyflow's pan-zoom controller, as the store holds it. */
type PanZoomInstance = NonNullable<ReactFlowState['panZoom']>

/** Renders the controls in a provider with a pan-zoom double at `zoom`; returns the store and the zoom spy. */
const mount = (zoom: number) => {
    const captured: { store?: Store } = {}
    const scaleBy = mock(async () => true)

    const Capture: FC = () => {
        captured.store = useStoreApi()
        return null
    }

    const view = render(
        <ReactFlowProvider>
            <Capture />
            <FlowControls insets={{ top: 10, right: 20, bottom: 30, left: 40 }}>
                <button type="button">Extra</button>
            </FlowControls>
        </ReactFlowProvider>,
    )
    const store = captured.store as Store
    act(() => store.setState({ width: 1100, height: 550, transform: [0, 0, zoom], panZoom: createPanZoom(scaleBy) }))

    return { ...view, store, scaleBy }
}

describe('FlowControls', () => {
    test('zooms in and out with an animation', () => {
        const { scaleBy, getByLabelText, getByText } = mount(1)

        fireEvent.click(getByLabelText('Zoom In'))
        fireEvent.click(getByLabelText('Zoom Out'))

        expect(scaleBy).toHaveBeenNthCalledWith(1, 1.2, { duration: ZOOM_DURATION })
        expect(scaleBy).toHaveBeenNthCalledWith(2, 1 / 1.2, { duration: ZOOM_DURATION })
        expect(getByText('Extra')).toBeDefined()
    })

    test('fits the view clear of the insets', () => {
        const { store, getByLabelText } = mount(1)

        fireEvent.click(getByLabelText('Fit View'))

        expect(store.getState().fitViewOptions).toEqual({
            duration: ZOOM_DURATION,
            padding: { top: '35px', right: '70px', bottom: '55px', left: '90px' },
        })
    })

    test('disables a zoom button at its limit', () => {
        expect(mount(2).getByLabelText('Zoom In').hasAttribute('disabled')).toBe(true)
        cleanup()

        expect(mount(0.5).getByLabelText('Zoom Out').hasAttribute('disabled')).toBe(true)
    })
})

/** A pan-zoom double whose `scaleBy` is `scaleBy`; every other method is inert. */
const createPanZoom = (scaleBy: PanZoomInstance['scaleBy']): PanZoomInstance => ({
    update: () => {},
    destroy: () => {},
    getViewport: () => ({ x: 0, y: 0, zoom: 1 }),
    setViewport: async () => undefined,
    setViewportConstrained: async () => undefined,
    setScaleExtent: () => {},
    setTranslateExtent: () => {},
    scaleTo: async () => true,
    scaleBy,
    syncViewport: () => {},
    setClickDistance: () => {},
})
