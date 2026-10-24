import { afterAll, beforeAll, describe, expect, test } from 'bun:test'

import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { importFresh } from '@/shared/testing/modules.fake'

// Loaded first so the fresh instance below is the one the coverage report keeps.
import { defaultFilterValues } from './store'

type StoreModule = typeof import('./store')

/** Key the filter persists under. */
const STORAGE_KEY = 'operations:filter'

/** An instance created while the DOM is in place, so it persists to `localStorage`. */
let fresh: StoreModule

beforeAll(async () => {
    registerDom()
    localStorage.clear()
    fresh = await importFresh<StoreModule>(import.meta.resolve('./store'))
})

afterAll(async () => {
    localStorage.clear()
    await unregisterDom()
})

/** What the store wrote to `localStorage`. */
const readPersisted = (): unknown => JSON.parse(localStorage.getItem(STORAGE_KEY) ?? 'null')

describe('useOperationsFilter persistence', () => {
    test('starts from the filter that allows everything', () => {
        expect(fresh.defaultFilterValues).toEqual(defaultFilterValues)
        expect(fresh.useOperationsFilter.getState()).toMatchObject(defaultFilterValues)
    })

    test('stores the six values and nothing else', () => {
        const state = fresh.useOperationsFilter.getState()
        state.setSinks(['kafka'])
        state.setTargets(['users'])
        state.setTables(['public.users'])
        state.setOperations(['insert'])
        state.setStatuses(['failed'])
        state.setSearch('boom')

        expect(readPersisted()).toEqual({
            state: {
                sinks: ['kafka'],
                targets: ['users'],
                tables: ['public.users'],
                operations: ['insert'],
                statuses: ['failed'],
                search: 'boom',
            },
            version: 1,
        })
    })

    test('applies several facets at once and keeps the others', () => {
        fresh.useOperationsFilter.getState().apply({ sinks: null, search: '' })

        expect(fresh.useOperationsFilter.getState()).toMatchObject({ sinks: null, search: '', targets: ['users'] })
    })
})
