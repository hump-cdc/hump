import { useLayoutEffect } from 'react'

import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, mock, spyOn, test } from 'bun:test'

import { act, cleanup, fireEvent, waitFor } from '@testing-library/react'

import type { SnapshotTableModel } from '@/shared/api'
import { type ApiHandler, empty, fakeApi, json, sse } from '@/shared/testing/api.fake'
import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { typeInto } from '@/shared/testing/input.fake'
import { flush, type ProvidersRender, renderWithProviders } from '@/shared/testing/render.fake'
import { toast } from '@/shared/ui/toast'

import { useTables } from '../../model/tables'
import { IdleActionBar, IdleView } from './index'

// Radix and Base UI pick their layout effect when first imported, as a no-op without a DOM; bun shares
// modules across test files, so point them at React's own one whatever order the files load in.
mock.module('@radix-ui/react-use-layout-effect', () => ({ useLayoutEffect }))
mock.module('@base-ui/utils/useIsoLayoutEffect', () => ({ useIsoLayoutEffect: useLayoutEffect }))
// prism-react-editor reads `document` once, when first loaded, which in a shared test process may be
// before any DOM exists; its caret tracker then breaks, so it is replaced with a no-op.
mock.module('prism-react-editor/cursor', () => ({ useCursorPosition: () => {} }))

/** A configured table as `POST /api/snapshot/tables` lists it. */
const tableModel = (table: string, where = ''): SnapshotTableModel => ({
    schema: 'public',
    table,
    type: 'regular',
    where,
    columns: [{ name: 'id', type: 'bigint', nullable: false, primaryKey: true }],
    sinks: [{ type: 'kafka', host: 'kafka:9092', target: table }],
})

/** The configured tables: `users` suggests a filter, the others read every row. */
const TABLES = [tableModel('users', 'id > 0'), tableModel('orders'), tableModel('events'), tableModel('audit')]

/** Counts `users` roughly and keeps counting, rejects `orders` and `events`, never answers for `audit`. */
const estimate: ApiHandler = (request, body) => {
    const [table] = (body as { tables: { table: string; where: string }[] }).tables
    switch (table?.table) {
        case 'users':
            return sse(request, [{ schema: 'public', table: 'users', count: 1_500, rough: true }], true)
        case 'orders':
        case 'events':
            return sse(request, [{ schema: 'public', table: table.table, error: `bad filter: ${table.where}` }])
        default:
            return sse(request, [], true)
    }
}

/** Counts every table exactly. */
const exactEstimate: ApiHandler = (request, body) => {
    const [table] = (body as { tables: { schema: string; table: string }[] }).tables
    return sse(request, [{ schema: 'public', table: table?.table, count: 10, rough: false }])
}

let view: ProvidersRender | undefined

beforeAll(registerDom)
afterAll(unregisterDom)

beforeEach(() => {
    useTables.setState({ filter: '', selectedIds: new Set(), wheres: {} })
    fakeApi.serve({
        'POST /api/snapshot/tables': () => json({ tables: TABLES }),
        'POST /api/snapshot/estimate': estimate,
    })
})

afterEach(async () => {
    await view?.client.cancelQueries()
    view?.client.clear()
    view = undefined
    cleanup()
})

/** Renders the idle view with its action bar. */
const mount = (): ProvidersRender => {
    view = renderWithProviders(
        <>
            <IdleView />
            <IdleActionBar />
        </>,
    )
    return view
}

/** Requests the page sent to `route`, with their bodies. */
const bodiesOf = (route: string): unknown[] =>
    fakeApi
        .calls()
        .filter((call) => call.route === route)
        .map(({ body }) => body)

describe('IdleView tables', () => {
    test('shows a spinner, then every configured table', async () => {
        const { getByText, findByText } = mount()

        expect(getByText('Loading tables…')).toBeDefined()
        expect(await findByText('public.audit')).toBeDefined()
        expect(getByText('When to use a snapshot')).toBeDefined()
    })

    test('explains a failed load', async () => {
        fakeApi.serve({ 'POST /api/snapshot/tables': () => json({ status: 500, detail: 'db is down' }, 500) })

        const { findByText } = mount()

        expect(await findByText('Failed to load tables')).toBeDefined()
        expect(await findByText('db is down')).toBeDefined()
    })

    test('says so when no table is configured', async () => {
        fakeApi.serve({ 'POST /api/snapshot/tables': () => json({ tables: [] }) })

        const { findByText } = mount()

        expect(await findByText('No configured tables found')).toBeDefined()
    })

    test('filters the tables by name', async () => {
        const { findByText, getByPlaceholderText, queryByText, getByText } = mount()
        await findByText('public.audit')

        act(() => typeInto(getByPlaceholderText('Filter tables…'), 'ORD'))
        expect(queryByText('public.audit')).toBeNull()
        expect(getByText('public.orders')).toBeDefined()

        act(() => typeInto(getByPlaceholderText('Filter tables…'), 'nothing'))
        expect(getByText(/No tables match/)).toBeDefined()
    })

    test('selects single tables and all visible ones', async () => {
        const { findByText, container } = mount()
        await findByText('public.audit')

        const usersBox = container.querySelector<HTMLElement>('#snapshot-table-public\\.users')
        const selectAll = container.querySelector<HTMLElement>('#snapshot-select-all')
        if (!usersBox || !selectAll) throw new Error('No checkboxes rendered')

        await act(async () => fireEvent.click(usersBox))
        expect([...useTables.getState().selectedIds]).toEqual(['public.users'])
        expect(selectAll.getAttribute('data-state')).toBe('indeterminate')

        await act(async () => fireEvent.click(selectAll))
        expect(useTables.getState().selectedIds.size).toBe(4)
        expect(selectAll.getAttribute('data-state')).toBe('checked')
    })
})

describe('IdleView estimates', () => {
    test('shows the count, the failures and the pending tables, and blocks the run', async () => {
        useTables.setState({ selectedIds: new Set(['public.users', 'public.orders', 'public.events', 'public.audit']) })

        const { findAllByText, findAllByLabelText, getByText, container } = mount()

        expect(await findAllByText(/^bad filter/)).toHaveLength(2)
        expect((await findAllByLabelText('Estimated from table statistics')).length).toBeGreaterThan(0)
        expect(container.querySelector('.animate-pulse')).not.toBeNull()
        expect(container.querySelector('.animate-spin')).not.toBeNull()
        expect(getByText(/filter errors/)).toBeDefined()
        expect(getByText(/^Fix the/).textContent).toContain('public.orders, public.events')
        expect((getByText('Run snapshot').closest('button') as HTMLButtonElement).disabled).toBe(true)
    })

    test('names a single failing table', async () => {
        useTables.setState({ selectedIds: new Set(['public.orders']) })

        const { findByText, getAllByText } = mount()

        expect(await findByText(/filter error$/)).toBeDefined()
        expect(getAllByText(/^table selected/)[0]).toBeDefined()
    })

    test('starts a run with the effective filter of every selected table', async () => {
        fakeApi.serve({
            'POST /api/snapshot/tables': () => json({ tables: TABLES }),
            'POST /api/snapshot/estimate': exactEstimate,
            'POST /api/snapshot/start': () => empty(202),
        })
        useTables.setState({ selectedIds: new Set(['public.users', 'public.orders']), wheres: { 'public.orders': '' } })

        const { findByText, getByText } = mount()
        expect(await findByText('Every row will be re-streamed to the connected sinks')).toBeDefined()
        const run = getByText('Run snapshot').closest('button') as HTMLButtonElement
        await waitFor(() => expect(run.disabled).toBe(false))

        await act(async () => fireEvent.click(run))

        await waitFor(() =>
            expect(bodiesOf('POST /api/snapshot/start')).toEqual([
                {
                    tables: [
                        { schema: 'public', table: 'users', where: 'id > 0' },
                        { schema: 'public', table: 'orders', where: '' },
                    ],
                },
            ]),
        )
    })

    test('reports a rejected start in a toast', async () => {
        const error = spyOn(toast, 'error').mockImplementation(() => 0)
        fakeApi.serve({
            'POST /api/snapshot/tables': () => json({ tables: TABLES }),
            'POST /api/snapshot/estimate': exactEstimate,
            'POST /api/snapshot/start': () => json({ status: 409, title: 'Conflict', detail: 'already running' }, 409),
        })
        useTables.setState({ selectedIds: new Set(['public.users']) })

        const { getByText, findByText } = mount()
        await findByText('public.users')
        const run = getByText('Run snapshot').closest('button') as HTMLButtonElement
        await waitFor(() => expect(run.disabled).toBe(false))
        await act(async () => fireEvent.click(run))

        await waitFor(() =>
            expect(error).toHaveBeenCalledWith('Failed to start the snapshot', { description: 'already running' }),
        )
        error.mockRestore()
    })
})

describe('IdleView row filter', () => {
    test('commits an edited filter on blur and follows a filter changed elsewhere', async () => {
        useTables.setState({ selectedIds: new Set(['public.orders']) })
        fakeApi.serve({
            'POST /api/snapshot/tables': () => json({ tables: TABLES }),
            'POST /api/snapshot/estimate': exactEstimate,
        })

        const { findByText, container } = mount()
        await findByText('public.orders')
        const textarea = container.querySelectorAll('textarea')[1]
        if (!textarea) throw new Error('No filter input for public.orders')

        await act(async () => {
            textarea.focus()
            textarea.value = 'id > 5'
            fireEvent.input(textarea)
        })
        await act(async () => textarea.blur())
        await flush()
        expect(useTables.getState().wheres).toEqual({ 'public.orders': 'id > 5' })

        act(() => useTables.getState().setWhere('public.orders', 'id > 9'))
        await flush()
        expect(textarea.value).toBe('id > 9')
    })
})
