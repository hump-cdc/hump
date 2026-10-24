import { useLayoutEffect } from 'react'

import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, type Mock, mock, spyOn, test } from 'bun:test'

import { act, cleanup, fireEvent, waitFor } from '@testing-library/react'

import { useFlowsFilterStore } from '@/entities/flows/flows/@x/toolbar'
import { fakeApi, json } from '@/shared/testing/api.fake'
import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { metricsResponse } from '@/shared/testing/fixtures.fake'
import { openComboBox } from '@/shared/testing/input.fake'
import { renderWithProviders } from '@/shared/testing/render.fake'

import { FlowsToolbar } from '../index'

// Radix and Base UI pick their layout effect when first imported, as a no-op without a DOM; bun shares
// modules across test files, so point them at React's own one whatever order the files load in.
mock.module('@radix-ui/react-use-layout-effect', () => ({ useLayoutEffect }))
mock.module('@base-ui/utils/useIsoLayoutEffect', () => ({ useIsoLayoutEffect: useLayoutEffect }))

beforeAll(registerDom)
afterAll(unregisterDom)

// The persist middleware warns when it cannot write; the warning is muted.
let warn: Mock<typeof console.warn>

beforeEach(() => {
    warn = spyOn(console, 'warn').mockImplementation(() => {})
    useFlowsFilterStore.setState({ tables: 'all', sinks: 'all' })
    fakeApi.serve({ 'POST /api/metrics': () => json(metricsResponse()) })
})

afterEach(() => {
    cleanup()
    useFlowsFilterStore.setState({ tables: 'all', sinks: 'all' })
    warn.mockRestore()
})

/** Renders the toolbar and waits for the options built from the metrics topology. */
const mount = async () => {
    const view = renderWithProviders(<FlowsToolbar />)
    await waitFor(() => expect(fakeApi.calls()).toHaveLength(1))
    return view
}

/** Combo boxes in the order the toolbar lays them out. */
const FACETS = ['Tables', 'Sinks'] as const

/** Toggles the option `name` in the combo box of `facet`. */
const toggleOption = async (
    view: Awaited<ReturnType<typeof mount>>,
    facet: (typeof FACETS)[number],
    name: string,
): Promise<void> => {
    // The placeholder shows only while nothing is picked, so the inputs are found by position.
    const input = view.getAllByRole('combobox')[FACETS.indexOf(facet)]
    if (!input) throw new Error(`No ${facet} combo box`)

    await openComboBox(input)
    await act(async () => fireEvent.click(await view.findByRole('option', { name: new RegExp(`^${name}`) })))
    await act(async () => fireEvent.keyDown(input, { key: 'Escape' }))
    await waitFor(() => expect(document.body.querySelector('[role="listbox"]')).toBeNull())
}

describe('FlowsToolbar', () => {
    test('narrows the tables and stores all once every one is picked again', async () => {
        const view = await mount()

        await toggleOption(view, 'Tables', 'public.users')
        expect(useFlowsFilterStore.getState().tables).toEqual([])

        await toggleOption(view, 'Tables', 'public.users')
        expect(useFlowsFilterStore.getState().tables).toBe('all')
    })

    test('lists every sink endpoint with its host and narrows a stored selection', async () => {
        useFlowsFilterStore.setState({ sinks: ['kafka|kafka:9092'] })
        const view = await mount()

        await toggleOption(view, 'Sinks', 'http · api.example.com')

        expect(useFlowsFilterStore.getState().sinks).toBe('all')

        await toggleOption(view, 'Sinks', 'kafka · kafka:9092')

        expect(useFlowsFilterStore.getState().sinks).toEqual(['http|api.example.com'])
    })
})
