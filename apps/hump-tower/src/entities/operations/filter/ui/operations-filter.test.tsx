import { useLayoutEffect } from 'react'

import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, type Mock, mock, spyOn, test } from 'bun:test'

import { act, cleanup, fireEvent, waitFor } from '@testing-library/react'

import { fakeApi, json } from '@/shared/testing/api.fake'
import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { configResponse, metricsResponse } from '@/shared/testing/fixtures.fake'
import { openComboBox, typeInto } from '@/shared/testing/input.fake'
import { renderWithProviders } from '@/shared/testing/render.fake'

import { useOperationsFilter as useListFilter } from '../@x/list'
import { defaultFilterValues, OperationsFilter, useOperationsFilter } from '../index'

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
    useOperationsFilter.setState(defaultFilterValues)
    fakeApi.serve({
        'GET /api/config': () => json(configResponse()),
        'POST /api/metrics': () => json(metricsResponse()),
    })
})

afterEach(() => {
    cleanup()
    warn.mockRestore()
})

/** Renders the filter and waits for the options built from the config and the metrics. */
const mount = async () => {
    const view = renderWithProviders(<OperationsFilter />)
    await waitFor(() => expect(fakeApi.calls()).toHaveLength(2))
    return view
}

/** Combo boxes in the order the filter lays them out. */
const FACETS = ['Tables', 'Sinks', 'Targets', 'Operations', 'Status'] as const

/** Toggles the option `name` in the combo box of `facet`. */
const toggleOption = async (
    view: Awaited<ReturnType<typeof mount>>,
    facet: (typeof FACETS)[number],
    name: string,
): Promise<void> => {
    const input = view.getAllByRole('combobox')[FACETS.indexOf(facet)]
    if (!input) throw new Error(`No ${facet} combo box`)

    await openComboBox(input)
    await act(async () => fireEvent.click(view.getByRole('option', { name: new RegExp(`^${name}`) })))
    await act(async () => fireEvent.keyDown(input, { key: 'Escape' }))
    await waitFor(() => expect(document.body.querySelector('[role="listbox"]')).toBeNull())
}

describe('OperationsFilter', () => {
    test('narrows a facet and folds a full selection back to all', async () => {
        const view = await mount()

        await toggleOption(view, 'Tables', 'public.orders')
        expect(useOperationsFilter.getState().tables).toEqual(['public.users'])

        await toggleOption(view, 'Tables', 'public.orders')
        expect(useOperationsFilter.getState().tables).toBeNull()
    })

    test('narrows the sinks and the targets the config and metrics offer', async () => {
        const view = await mount()

        await toggleOption(view, 'Sinks', 'http')
        await toggleOption(view, 'Targets', 'users')

        expect(useOperationsFilter.getState().sinks).toEqual(['kafka'])
        expect(useOperationsFilter.getState().targets).toEqual([])
    })

    test('narrows the operations and statuses and sets the search text', async () => {
        const view = await mount()

        await toggleOption(view, 'Operations', 'truncate')
        await toggleOption(view, 'Status', 'failed')
        act(() => typeInto(view.getByPlaceholderText('Search payload, target, error…'), 'boom'))

        const state = useListFilter.getState()
        expect(state.operations).toEqual(['insert', 'update', 'delete'])
        expect(state.statuses).toEqual(['delivered'])
        expect(state.search).toBe('boom')
    })

    test('offers no options before the config and metrics arrive', async () => {
        fakeApi.serve({})

        const view = renderWithProviders(<OperationsFilter />)
        await openComboBox(view.getByPlaceholderText('Tables'))
        await waitFor(() => expect(fakeApi.calls()).toHaveLength(2))

        expect(view.queryAllByRole('option')).toHaveLength(0)
    })
})
