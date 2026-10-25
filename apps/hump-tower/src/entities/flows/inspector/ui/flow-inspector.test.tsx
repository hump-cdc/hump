import { useLayoutEffect } from 'react'

import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, mock, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { act, cleanup, fireEvent } from '@testing-library/react'

import { fakeApi, json } from '@/shared/testing/api.fake'
import { stubMatchMedia } from '@/shared/testing/browser-stubs.fake'
import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { configResponse, metricsResponse, operationItem, operationsResponse } from '@/shared/testing/fixtures.fake'
import { fakeLiveStream } from '@/shared/testing/live.fake'
import { flush, renderWithRouter } from '@/shared/testing/render.fake'

import { useFlowInspectorStore as useCanvasInspectorStore } from '../@x/flows'
import { FlowInspector, useFlowInspectorStore } from '../index'

// Radix picks its layout effect when first imported, as a no-op without a DOM; bun shares modules
// across test files, so point it at React's own one whatever order the files load in.
mock.module('@radix-ui/react-use-layout-effect', () => ({ useLayoutEffect }))

const SINK = { type: 'kafka', host: 'kafka:9092' }

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) {
        GlobalRegistrator.register({
            url: 'http://localhost/',
            settings: { device: { prefersReducedMotion: 'reduce' } },
        })
    }
    registerDom()
})
afterAll(unregisterDom)
beforeEach(() => {
    fakeLiveStream.reset()
    useFlowInspectorStore.setState(useFlowInspectorStore.getInitialState(), true)
    fakeApi.serve({
        'POST /api/metrics': () => json(metricsResponse()),
        'GET /api/config': () => json(configResponse()),
        'POST /api/operations': () => json(operationsResponse([operationItem(1)])),
    })
})
afterEach(cleanup)

/** The sheet panel, while it is open. */
const getPanel = () => document.body.querySelector('[data-slot="sheet-content"]')

describe('FlowInspector', () => {
    test('stays closed without a selection', async () => {
        await renderWithRouter(<FlowInspector />)

        expect(getPanel()).toBeNull()
    })

    test('opens on a canvas selection with its recent operations and a link to the operations', async () => {
        const view = await renderWithRouter(<FlowInspector />)

        await act(async () =>
            useCanvasInspectorStore.getState().selectTable({ id: 'public.users', name: 'public.users' }),
        )

        expect(getPanel()?.className).toContain('right-0')
        expect(view.getByText('Table details')).toBeDefined()
        expect(view.getByText('RECENT OPERATIONS')).toBeDefined()
        expect(await view.findByText('INSERT')).toBeDefined()
        const link = view.getByText('View in Operations').closest('a')
        expect(link?.getAttribute('href')).toBe('/operations?table=public.users')
    })

    test('titles each selection kind', async () => {
        const view = await renderWithRouter(<FlowInspector />)
        const { selectSink, selectTarget, selectEdge } = useFlowInspectorStore.getState()

        await act(async () => selectSink({ id: 'kafka', name: 'Kafka', ...SINK }))
        expect(view.getByText('Sink details')).toBeDefined()

        await act(async () => selectTarget({ id: 'kafka/users', name: 'users', sinkLabel: 'Kafka', ...SINK }))
        expect(view.getByText('Sink details')).toBeDefined()

        await act(async () =>
            selectEdge({ id: 'e', sourceLabel: 'users', targetLabel: 'kafka', table: 'public.users', ...SINK }),
        )
        expect(view.getByText('Connection details')).toBeDefined()
    })

    test('stays open on an interaction outside and clears the selection on close', async () => {
        useFlowInspectorStore.getState().selectTable({ id: 'public.users', name: 'public.users' })
        const view = await renderWithRouter(<FlowInspector />)
        await flush()

        await act(async () => {
            fireEvent.pointerDown(document.body)
            fireEvent.focusIn(document.body)
        })
        expect(useFlowInspectorStore.getState().selected).not.toBeNull()

        await act(async () => fireEvent.click(view.getByLabelText('Close inspector')))
        expect(useFlowInspectorStore.getState().selected).toBeNull()
    })

    test('docks to the bottom on mobile without the recent operations', async () => {
        const restore = stubMatchMedia(true)
        try {
            useFlowInspectorStore.getState().selectTable({ id: 'public.users', name: 'public.users' })

            const view = await renderWithRouter(<FlowInspector />)

            expect(getPanel()?.className).toContain('bottom-0')
            expect(view.queryByText('RECENT OPERATIONS')).toBeNull()
        } finally {
            restore()
        }
    })
})
