import type { FC } from 'react'

import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, type Mock, spyOn, test } from 'bun:test'

import { act, cleanup, fireEvent, waitFor } from '@testing-library/react'

import { useFlowInspectorStore } from '@/entities/flows/inspector/@x/flows'
import { fakeApi, json } from '@/shared/testing/api.fake'
import { stubLayout, stubResizeObserver } from '@/shared/testing/browser-stubs.fake'
import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { metricsResponse } from '@/shared/testing/fixtures.fake'
import { fakeLiveStream } from '@/shared/testing/live.fake'
import { flush, renderWithProviders } from '@/shared/testing/render.fake'

import { useFlowsTopology } from '../@x/toolbar'
import { Flows } from '../index'
import { useFlowsFilterStore } from '../model/filter'
import { useTopologyModeStore } from '../model/mode'

// Type-only module: loaded for its side effects so the coverage report lists it.
import '../model/types'

const TABLE = 'public.users'
const KAFKA = 'kafka|kafka:9092'

/** Restores what `beforeAll` stubbed. */
let restoreLayout: () => void
let restoreResizeObserver: () => void

beforeAll(() => {
    registerDom()
    // React Flow draws an edge only once both of its nodes are measured: give every element a box and
    // report it as soon as an element is observed.
    restoreLayout = stubLayout(200, 100)
    restoreResizeObserver = stubResizeObserver()
})

afterAll(async () => {
    restoreResizeObserver()
    restoreLayout()
    await unregisterDom()
})

// The persist middleware warns when it cannot write; the warning is muted.
let warn: Mock<typeof console.warn>

beforeEach(() => {
    warn = spyOn(console, 'warn').mockImplementation(() => {})
    useTopologyModeStore.setState({ mode: 'detailed' })
    useFlowsFilterStore.setState({ tables: 'all', sinks: 'all' })
    useFlowInspectorStore.getState().clear()
    fakeApi.serve({ 'POST /api/metrics': () => json(metricsResponse()) })
})

afterEach(() => {
    cleanup()
    fakeLiveStream.reset()
    useTopologyModeStore.setState({ mode: 'detailed' })
    useFlowInspectorStore.getState().clear()
    warn.mockRestore()
})

/** Renders the canvas and waits until its edges are drawn. */
const mount = async () => {
    const view = renderWithProviders(<Flows overlay={<div>inspector</div>} />)
    await view.findByText(TABLE)
    await flush(10)
    await waitFor(() => expect(view.container.querySelector('.react-flow__edge')).not.toBeNull())
    return view
}

/** The React Flow element of `selector` inside the layer that is not fading out. */
const activeElement = (container: HTMLElement, selector: string): Element => {
    const element = container.querySelector(`div:not([inert]) > * ${selector}`)
    if (!element) throw new Error(`${selector} is not rendered`)
    return element
}

/** Clicks `selector` in the active layer. */
const click = (container: HTMLElement, selector: string): void => {
    act(() => {
        fireEvent.click(activeElement(container, selector))
    })
}

/** What the inspector store holds now. */
const selected = () => useFlowInspectorStore.getState().selected

describe('Flows in the detailed view', () => {
    test('renders the overlay and the topology', async () => {
        const { getByText } = await mount()

        expect(getByText('inspector')).toBeDefined()
        expect(getByText('users')).toBeDefined()
        expect(getByText('kafka:9092')).toBeDefined()
    })

    test('selects the clicked table, sink, target and edge, and clears on a pane click', async () => {
        const { container, getByText } = await mount()

        click(container, `[data-id="${TABLE}"]`)
        expect(selected()).toMatchObject({ kind: 'table', id: TABLE, name: TABLE })

        click(container, `[data-id="${KAFKA}"]`)
        expect(selected()).toMatchObject({ kind: 'sink', id: KAFKA, type: 'kafka', host: 'kafka:9092' })

        act(() => {
            fireEvent.click(getByText('users'))
        })
        expect(selected()).toMatchObject({ kind: 'target', name: 'users', sinkLabel: 'kafka', host: 'kafka:9092' })

        click(container, '.react-flow__edge')
        expect(selected()).toEqual({
            kind: 'edge',
            id: `${TABLE}>${KAFKA}>users`,
            sourceLabel: TABLE,
            targetLabel: 'kafka/users',
            table: TABLE,
            type: 'kafka',
            host: 'kafka:9092',
            target: 'users',
        })

        click(container, '.react-flow__pane')
        expect(selected()).toBeNull()
    })

    test('flashes the edge of every live delivery', async () => {
        await mount()

        expect(fakeLiveStream.listeners()).toBe(1)
        act(() => {
            fakeLiveStream.emit([
                { schema: 'public', table: 'users', sink: 'kafka', host: 'kafka:9092', target: 'users' },
            ])
        })
        await flush()

        expect(fakeLiveStream.listeners()).toBe(1)
    })

    test('hides what the toolbar filtered out', async () => {
        useFlowsFilterStore.setState({ sinks: ['http|api.example.com'] })

        const { findByText, queryByText } = renderWithProviders(<Flows />)

        expect(await findByText('api.example.com')).toBeDefined()
        expect(queryByText('kafka:9092')).toBeNull()
    })
})

describe('Flows mode switch', () => {
    test('clears the selection, cross-fades to the summary view and drops the old view after its fade', async () => {
        const { container, getAllByRole } = await mount()
        click(container, `[data-id="${TABLE}"]`)

        const [summary] = getAllByRole('button', { name: 'Summary' })
        if (!summary) throw new Error('no Summary button')
        expect(summary.getAttribute('aria-pressed')).toBe('false')
        act(() => {
            fireEvent.click(summary)
        })

        expect(useTopologyModeStore.getState().mode).toBe('summary')
        expect(selected()).toBeNull()
        const leaving = container.querySelector('[inert]')
        if (!leaving) throw new Error('the detailed view is not fading out')
        expect(leaving.className).toContain('animate-out')
        expect(container.querySelector('.animate-in')).not.toBeNull()

        // An animation bubbling up from inside the layer does not end the transition.
        act(() => {
            fireEvent.animationEnd(activeElement(container, '.react-flow'))
            fireEvent.animationEnd(container.querySelector('.animate-in') ?? leaving)
            fireEvent.animationEnd(leaving.firstElementChild ?? leaving)
        })
        expect(container.querySelector('[inert]')).toBe(leaving)

        act(() => {
            fireEvent.animationEnd(leaving)
        })
        expect(container.querySelector('[inert]')).toBeNull()
        expect(getAllByRole('button', { name: 'Summary' })).toHaveLength(1)
    })

    test('ignores a click on the mode already shown', async () => {
        const { container, getByRole } = await mount()
        click(container, `[data-id="${TABLE}"]`)

        act(() => {
            fireEvent.click(getByRole('button', { name: 'Detailed' }))
        })

        expect(useTopologyModeStore.getState().mode).toBe('detailed')
        expect(selected()).toMatchObject({ kind: 'table' })
    })
})

describe('Flows in the summary view', () => {
    test('selects the clicked table, sink and edge, clears on a pane click and flashes live deliveries', async () => {
        useTopologyModeStore.setState({ mode: 'summary' })
        const { container } = await mount()

        click(container, `[data-id="${TABLE}"]`)
        expect(selected()).toMatchObject({ kind: 'table', id: TABLE })

        click(container, `[data-id="${KAFKA}"]`)
        expect(selected()).toMatchObject({ kind: 'sink', id: KAFKA, host: 'kafka:9092' })

        click(container, '.react-flow__edge')
        expect(selected()).toEqual({
            kind: 'edge',
            id: `${TABLE}>${KAFKA}`,
            sourceLabel: TABLE,
            targetLabel: 'kafka',
            table: TABLE,
            type: 'kafka',
            host: 'kafka:9092',
        })

        click(container, '.react-flow__pane')
        expect(selected()).toBeNull()

        act(() => {
            fakeLiveStream.emit([
                { schema: 'public', table: 'users', sink: 'kafka', host: 'kafka:9092', target: 'users' },
            ])
        })
        await flush()
        expect(fakeLiveStream.listeners()).toBe(1)
    })
})

describe('toolbar cross-import', () => {
    test('exposes the topology the canvas renders', async () => {
        const { findByText } = renderWithProviders(<TopologyProbe />)

        expect(await findByText('tables: public.users')).toBeDefined()
    })
})

/** Prints the table ids `useFlowsTopology` returns through the toolbar's `@x` door. */
const TopologyProbe: FC = () => (
    <span>
        tables:{' '}
        {useFlowsTopology()
            .tables.map(({ id }) => id)
            .join(',')}
    </span>
)
