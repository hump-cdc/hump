import type { FC, RefObject } from 'react'

import { afterAll, afterEach, beforeAll, describe, expect, mock, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { act, cleanup, render } from '@testing-library/react'
import {
    type Edge,
    type Handle,
    type InternalNode,
    Position,
    ReactFlowProvider,
    type ReactFlowState,
    useStoreApi,
} from '@xyflow/react'

import { REVEAL_DURATION, REVEAL_MARGIN } from './flow.consts'
import { FlowReveal } from './flow.reveal'
import type { ViewportInsets } from './flow.types'

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(cleanup)

const noInsets: ViewportInsets = { top: 0, right: 0, bottom: 0, left: 0 }

type Store = ReturnType<typeof useStoreApi>

/** xyflow's pan-zoom controller, as the store holds it. */
type PanZoomInstance = NonNullable<ReactFlowState['panZoom']>

type RevealProps = { revealId?: string; revealKey?: string; insets?: ViewportInsets; userMoved?: RefObject<boolean> }

/** Renders `FlowReveal` in a provider and hands back the provider's store and a viewport spy. */
const mount = (props: RevealProps) => {
    const captured: { store?: Store } = {}
    const setViewport = mock(async () => undefined)
    const userMovedRef = props.userMoved ?? { current: false }

    const Capture: FC = () => {
        captured.store = useStoreApi()
        return null
    }

    const tree = (next: RevealProps) => (
        <ReactFlowProvider>
            <Capture />
            <FlowReveal
                revealId={next.revealId}
                revealKey={next.revealKey ?? next.revealId}
                insets={next.insets ?? noInsets}
                userMovedRef={userMovedRef}
            />
        </ReactFlowProvider>
    )
    const view = render(tree(props))
    const store = captured.store as Store
    act(() => store.setState({ width: 1000, height: 800, transform: [0, 0, 1], panZoom: createPanZoom(setViewport) }))

    return { store, setViewport, userMovedRef, rerender: (next: RevealProps) => view.rerender(tree(next)) }
}

/** Waits one animation frame, so the reveal deferred by `FlowReveal` has run. */
const nextFrame = (): Promise<void> => act(() => new Promise<void>((resolve) => requestAnimationFrame(() => resolve())))

describe('FlowReveal', () => {
    test('pans a measured node outside the area into view', async () => {
        const { store, setViewport } = mount({ revealId: 'n' })
        act(() => store.setState({ nodeLookup: new Map([['n', internalNode('n', { x: 2000, y: 100 }, [])]]) }))

        await nextFrame()

        expect(setViewport).toHaveBeenCalledWith(
            { x: 1000 - REVEAL_MARGIN - 2200, y: 0, zoom: 1 },
            expect.objectContaining({ duration: REVEAL_DURATION }),
        )
    })

    test('stays put when the node is already inside the area', async () => {
        const { store, setViewport } = mount({ revealId: 'n' })
        act(() => store.setState({ nodeLookup: new Map([['n', internalNode('n', { x: 100, y: 100 }, [])]]) }))

        await nextFrame()

        expect(setViewport).not.toHaveBeenCalled()
    })

    test('waits for a node to be measured', async () => {
        const { store, setViewport } = mount({ revealId: 'n' })
        const node = internalNode('n', { x: 2000, y: 0 }, [])
        act(() => store.setState({ nodeLookup: new Map([['n', { ...node, measured: {} }]]) }))

        await nextFrame()

        expect(setViewport).not.toHaveBeenCalled()
    })

    test('reveals the span between an edge’s handles', async () => {
        const { store, setViewport } = mount({ revealId: 'e' })
        const source = internalNode('a', { x: 0, y: 0 }, [handle('a', 'source', 'out', 90, 10)])
        const target = internalNode('b', { x: 1500, y: 900 }, [
            handle('b', 'target', 'other', 0, 50),
            handle('b', 'target', 'in', 0, 10),
        ])
        const edge: Edge = { id: 'e', source: 'a', target: 'b', targetHandle: 'in' }
        act(() =>
            store.setState({
                nodeLookup: new Map([
                    ['a', source],
                    ['b', target],
                ]),
                edgeLookup: new Map([['e', edge]]),
            }),
        )

        await nextFrame()

        expect(setViewport).toHaveBeenCalledWith(
            { x: REVEAL_MARGIN - 95, y: REVEAL_MARGIN - 15, zoom: 1 },
            expect.objectContaining({ duration: REVEAL_DURATION }),
        )
    })

    test('skips an edge whose handles are not measured', async () => {
        const { store, setViewport } = mount({ revealId: 'e' })
        act(() =>
            store.setState({
                nodeLookup: new Map([['a', internalNode('a', { x: 0, y: 0 }, [])]]),
                edgeLookup: new Map([['e', { id: 'e', source: 'a', target: 'missing' }]]),
            }),
        )

        await nextFrame()

        expect(setViewport).not.toHaveBeenCalled()
    })

    test('skips an unknown id', async () => {
        const { setViewport } = mount({ revealId: 'unknown' })

        await nextFrame()

        expect(setViewport).not.toHaveBeenCalled()
    })

    test('keeps the reveal clear of the zoom controls', async () => {
        const { store, setViewport } = mount({ revealId: 'n' })
        const domNode = document.createElement('div')
        const controls = document.createElement('div')
        controls.className = 'react-flow__controls'
        domNode.append(controls)
        controls.getBoundingClientRect = () => DOMRect.fromRect({ x: 0, y: 0, width: 300, height: 10 })
        act(() => store.setState({ domNode, nodeLookup: new Map([['n', internalNode('n', { x: 100, y: 100 }, [])]]) }))

        await nextFrame()

        expect(setViewport).toHaveBeenCalledWith(
            { x: 300 + REVEAL_MARGIN - 100, y: 0, zoom: 1 },
            expect.objectContaining({ duration: REVEAL_DURATION }),
        )
    })

    test('does nothing without a reveal id or after the user moved, until the key changes', async () => {
        const userMoved = { current: true }
        const { store, setViewport, rerender, userMovedRef } = mount({ revealId: undefined, userMoved })
        act(() => store.setState({ nodeLookup: new Map([['n', internalNode('n', { x: 2000, y: 0 }, [])]]) }))
        await nextFrame()
        expect(setViewport).not.toHaveBeenCalled()

        rerender({ revealId: 'n', revealKey: undefined })
        userMovedRef.current = true
        rerender({ revealId: 'n', revealKey: undefined, insets: { ...noInsets, top: 1 } })
        await nextFrame()
        expect(setViewport).not.toHaveBeenCalled()

        rerender({ revealId: 'n', revealKey: 'k' })
        await nextFrame()
        expect(userMovedRef.current).toBe(false)
        expect(setViewport).toHaveBeenCalledTimes(1)
    })
})

/** A measured internal node at `position`, 200×100, with the given handles. */
const internalNode = (id: string, position: { x: number; y: number }, handles: Handle[]): InternalNode => {
    const node = { id, position, data: {} }
    return {
        ...node,
        measured: { width: 200, height: 100 },
        internals: {
            positionAbsolute: position,
            z: 0,
            userNode: node,
            handleBounds: {
                source: handles.filter((item) => item.type === 'source'),
                target: handles.filter((item) => item.type === 'target'),
            },
        },
    }
}

/** A 10×10 handle of `nodeId` at node-relative (`x`, `y`). */
const handle = (nodeId: string, type: 'source' | 'target', id: string, x: number, y: number): Handle => ({
    id,
    nodeId,
    type,
    x,
    y,
    width: 10,
    height: 10,
    position: type === 'source' ? Position.Right : Position.Left,
})

/** A pan-zoom double whose `setViewport` is `setViewport`; every other method is inert. */
const createPanZoom = (setViewport: PanZoomInstance['setViewport']): PanZoomInstance => ({
    update: () => {},
    destroy: () => {},
    getViewport: () => ({ x: 0, y: 0, zoom: 1 }),
    setViewport,
    setViewportConstrained: async () => undefined,
    setScaleExtent: () => {},
    setTranslateExtent: () => {},
    scaleTo: async () => true,
    scaleBy: async () => true,
    syncViewport: () => {},
    setClickDistance: () => {},
})
