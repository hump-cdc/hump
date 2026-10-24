import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, type Mock, spyOn, test } from 'bun:test'

import { act, cleanup, waitFor } from '@testing-library/react'

import type { SnapshotTableModel } from '@/shared/api'
import { fakeApi, json, sse } from '@/shared/testing/api.fake'
import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { importFresh } from '@/shared/testing/modules.fake'
import { renderHookWithProviders } from '@/shared/testing/render.fake'

// Loaded first so the fresh instance below is the one the coverage report keeps.
import './tables'

type TablesModule = typeof import('./tables')

/** Key the store persists under. */
const STORAGE_KEY = 'snapshot-tables'

/** A configured table as `POST /api/snapshot/tables` lists it. */
const tableModel = (table: string, overrides: Partial<SnapshotTableModel> = {}): SnapshotTableModel => ({
    schema: 'public',
    table,
    type: 'regular',
    where: '',
    columns: [{ name: 'id', type: 'bigint', nullable: false, primaryKey: true }],
    sinks: [{ type: 'kafka', host: 'kafka:9092', target: table }],
    ...overrides,
})

/** The store and hooks, from an instance that binds to the `localStorage` of the registered DOM. */
let tables: TablesModule

beforeAll(async () => {
    registerDom()
    localStorage.clear()
    localStorage.setItem(
        STORAGE_KEY,
        JSON.stringify({ state: { selectedIds: ['public.users'], wheres: { 'public.users': 'id > 1' } }, version: 1 }),
    )
    tables = await importFresh<TablesModule>(import.meta.resolve('./tables'))
})

afterAll(async () => {
    localStorage.clear()
    await unregisterDom()
})

/** What the store wrote to `localStorage`. */
const readPersisted = (): unknown => JSON.parse(localStorage.getItem(STORAGE_KEY) ?? 'null')

describe('useTables persistence', () => {
    test('restores the selection as a set and the filters on load', () => {
        const { selectedIds, wheres, filter } = tables.useTables.getState()

        expect([...selectedIds]).toEqual(['public.users'])
        expect(wheres).toEqual({ 'public.users': 'id > 1' })
        expect(filter).toBe('')
    })

    test('stores the selection as an array and leaves the name filter out', () => {
        tables.useTables.getState().setFilter('ord')
        tables.useTables.getState().toggle('public.orders')

        expect(readPersisted()).toEqual({
            state: { selectedIds: ['public.users', 'public.orders'], wheres: { 'public.users': 'id > 1' } },
            version: 1,
        })
    })

    test('falls back to an empty selection when nothing usable is stored', async () => {
        localStorage.setItem(STORAGE_KEY, JSON.stringify({ state: {}, version: 1 }))
        await tables.useTables.persist.rehydrate()

        expect(tables.useTables.getState().selectedIds.size).toBe(0)
        expect(tables.useTables.getState().wheres).toEqual({})
    })
})

describe('useTables', () => {
    beforeEach(() => {
        tables.useTables.setState({ filter: '', selectedIds: new Set(), wheres: {} })
    })

    test('sets the name filter', () => {
        tables.useTables.getState().setFilter('orders')

        expect(tables.useTables.getState().filter).toBe('orders')
    })

    test('toggles a single table', () => {
        tables.useTables.getState().toggle('public.orders')
        expect([...tables.useTables.getState().selectedIds]).toEqual(['public.orders'])

        tables.useTables.getState().toggle('public.orders')
        expect(tables.useTables.getState().selectedIds.size).toBe(0)
    })

    test('selects all given tables unless every one is already selected', () => {
        tables.useTables.getState().toggle('public.users')

        tables.useTables.getState().toggleAll(['public.users', 'public.orders'])
        expect([...tables.useTables.getState().selectedIds]).toEqual(['public.users', 'public.orders'])

        tables.useTables.getState().toggleAll(['public.users', 'public.orders'])
        expect(tables.useTables.getState().selectedIds.size).toBe(0)
    })

    test('keeps tables outside the given ids selected', () => {
        tables.useTables.getState().toggle('audit.log')
        tables.useTables.getState().toggleAll(['public.users'])
        tables.useTables.getState().toggleAll(['public.users'])

        expect([...tables.useTables.getState().selectedIds]).toEqual(['audit.log'])
    })

    test('sets a filter per table, keeping the state when it is unchanged', () => {
        tables.useTables.getState().setWhere('public.users', 'id > 1')
        const state = tables.useTables.getState()

        tables.useTables.getState().setWhere('public.users', 'id > 1')
        expect(tables.useTables.getState()).toBe(state)

        tables.useTables.getState().setWhere('public.users', '')
        expect(tables.useTables.getState().wheres).toEqual({ 'public.users': '' })
    })
})

describe('useAvailableTables and useSelectedTables', () => {
    beforeEach(() => {
        tables.useTables.setState({ filter: '', selectedIds: new Set(['public.orders']), wheres: {} })
        fakeApi.serve({
            'POST /api/snapshot/tables': () =>
                json({
                    tables: [
                        tableModel('users', { where: 'id > 0' }),
                        tableModel('orders', { columns: null, sinks: null }),
                    ],
                }),
        })
    })
    afterEach(cleanup)

    test('maps the configured tables with their columns, sinks and suggested filter', async () => {
        const { result } = renderHookWithProviders(() => tables.useAvailableTables())

        await waitFor(() => expect(result.current.data).toHaveLength(2))
        expect(result.current.data).toEqual([
            {
                id: 'public.users',
                schema: 'public',
                table: 'users',
                name: 'public.users',
                suggestedWhere: 'id > 0',
                columns: [{ name: 'id', type: 'bigint' }],
                sinks: [{ type: 'kafka', host: 'kafka:9092', target: 'users' }],
            },
            {
                id: 'public.orders',
                schema: 'public',
                table: 'orders',
                name: 'public.orders',
                suggestedWhere: '',
                columns: [],
                sinks: [],
            },
        ])
    })

    test('lists only the selected tables', async () => {
        const { result } = renderHookWithProviders(() => tables.useSelectedTables())

        await waitFor(() => expect(result.current.map(({ id }) => id)).toEqual(['public.orders']))
    })

    test('lists nothing when the response has no tables', async () => {
        fakeApi.serve({ 'POST /api/snapshot/tables': () => json({ tables: null }) })

        const { result } = renderHookWithProviders(() => tables.useAvailableTables())

        await waitFor(() => expect(result.current.data).toEqual([]))
    })
})

describe('useTablesEstimate', () => {
    // A rejected stream payload is logged; the log is muted and checked.
    let warn: Mock<typeof console.warn>

    beforeEach(() => {
        warn = spyOn(console, 'warn').mockImplementation(() => {})
        tables.useTables.setState({
            filter: '',
            selectedIds: new Set(['public.users', 'public.orders', 'public.events', 'public.audit']),
            wheres: { 'public.orders': 'bad' },
        })
        fakeApi.serve({
            'POST /api/snapshot/tables': () =>
                json({
                    tables: [tableModel('users'), tableModel('orders'), tableModel('events'), tableModel('audit')],
                }),
            'POST /api/snapshot/estimate': (request, body) => {
                const [table] = (body as { tables: { table: string; where: string }[] }).tables
                switch (table?.table) {
                    case 'users':
                        return sse(request, [
                            { schema: 'public', table: 'users', count: 900, rough: true },
                            { schema: 'public', table: 'users', count: 1000, rough: false },
                            {},
                        ])
                    case 'orders':
                        return sse(request, [
                            { schema: 'public', table: 'orders', error: `invalid filter: ${table.where}` },
                        ])
                    case 'events':
                        return sse(request, ['{"count":"not a number"}', {}])
                    default:
                        return sse(request, [], true)
                }
            },
        })
    })
    afterEach(() => {
        cleanup()
        warn.mockRestore()
    })

    test('counts each selected table under its filter and reports the failures', async () => {
        const { result } = renderHookWithProviders(() => tables.useTablesEstimate())

        await waitFor(() =>
            expect(result.current).toEqual({
                'public.users': { status: 'ready', count: 1000, rough: false, fetching: false },
                'public.orders': { status: 'error', error: 'invalid filter: bad' },
                'public.events': { status: 'error', error: 'No estimate received' },
                'public.audit': { status: 'loading' },
            }),
        )
        expect(warn).toHaveBeenCalledWith('[snapshot] estimate stream error', expect.anything())
    })

    test('stays quiet about a stream it aborted itself', async () => {
        const { unmount } = renderHookWithProviders(() => tables.useTablesEstimate())
        await waitFor(() => expect(fakeApi.calls().filter(({ route }) => route.endsWith('estimate'))).toHaveLength(4))
        warn.mockClear()

        act(() => unmount())

        expect(warn).not.toHaveBeenCalledWith('[snapshot] estimate stream error', expect.anything())
    })
})
