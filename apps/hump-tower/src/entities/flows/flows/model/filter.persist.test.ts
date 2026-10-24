import { afterAll, beforeAll, describe, expect, test } from 'bun:test'

import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { importFresh } from '@/shared/testing/modules.fake'

// Loaded first so the fresh instance below is the one the coverage report keeps.
import { useFlowsFilterStore } from './filter'
import { type FlowsTopology, sinkId } from './topology'

type FilterModule = typeof import('./filter')

/** Key the filter persists under. */
const STORAGE_KEY = 'hump.flows-filter'

const KAFKA = sinkId('kafka', 'kafka:9092')
const HTTP = sinkId('http', 'api.example.com')

/** Operation counts of a table that received nothing. */
const IDLE = { count: 0 }

/** One table delivered to two sinks, in both views. */
const TOPOLOGY: FlowsTopology = {
    tables: [
        { id: 'public.users', name: 'public.users', ops: { insert: IDLE, update: IDLE, delete: IDLE, truncate: IDLE } },
    ],
    sinks: [
        { id: KAFKA, name: 'kafka', type: 'kafka', operations: 1, health: 'ok' },
        { id: HTTP, name: 'http', type: 'http', operations: 1, health: 'ok' },
    ],
    edges: [
        { id: `public.users>${KAFKA}`, source: 'public.users', target: KAFKA },
        { id: `public.users>${HTTP}`, source: 'public.users', target: HTTP },
    ],
    detailedSinks: [
        { id: KAFKA, name: 'kafka', type: 'kafka', count: 1, targets: [] },
        { id: HTTP, name: 'http', type: 'http', count: 1, targets: [] },
    ],
    detailedEdges: [{ id: `public.users>${KAFKA}>users`, source: 'public.users', target: KAFKA }],
}

/** An instance created while the DOM is in place, so it persists to `localStorage`. */
let fresh: FilterModule

beforeAll(async () => {
    registerDom()
    localStorage.clear()
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ state: { tables: 'bogus', sinks: [KAFKA] }, version: 0 }))
    fresh = await importFresh<FilterModule>(import.meta.resolve('./filter'))
    // Hydrated again explicitly, so the stored version-0 state goes through `migrate` before the tests.
    await fresh.useFlowsFilterStore.persist.rehydrate()
})

afterAll(async () => {
    localStorage.clear()
    await unregisterDom()
})

/** What the store wrote to `localStorage`. */
const readPersisted = (): unknown => JSON.parse(localStorage.getItem(STORAGE_KEY) ?? 'null')

describe('useFlowsFilterStore persistence', () => {
    test('migrates a selection stored by an older version', () => {
        expect(useFlowsFilterStore).not.toBe(fresh.useFlowsFilterStore)
        expect(fresh.useFlowsFilterStore.getState()).toMatchObject({ tables: 'all', sinks: [KAFKA] })
    })

    test('stores the two selections and nothing else', () => {
        fresh.useFlowsFilterStore.getState().setTables(['public.users'])
        fresh.useFlowsFilterStore.getState().setSinks([HTTP])

        expect(readPersisted()).toEqual({ state: { tables: ['public.users'], sinks: [HTTP] }, version: 1 })
    })
})

describe('the fresh instance', () => {
    test('migrateFlowsFilter keeps lists of ids and drops anything else', () => {
        expect(fresh.migrateFlowsFilter({ tables: ['a'], sinks: [1] })).toEqual({ tables: ['a'], sinks: 'all' })
        expect(fresh.migrateFlowsFilter(null)).toEqual({ tables: 'all', sinks: 'all' })
    })

    test('filterFlowsTopology narrows both views', () => {
        expect(fresh.filterFlowsTopology(TOPOLOGY, 'all', 'all')).toBe(TOPOLOGY)

        const narrowed = fresh.filterFlowsTopology(TOPOLOGY, 'all', [KAFKA])
        expect(narrowed.sinks.map(({ id }) => id)).toEqual([KAFKA])
        expect(narrowed.edges.map(({ target }) => target)).toEqual([KAFKA])
        expect(narrowed.detailedSinks.map(({ id }) => id)).toEqual([KAFKA])
        expect(narrowed.detailedEdges).toHaveLength(1)
        expect(fresh.filterFlowsTopology(TOPOLOGY, [], 'all').tables).toEqual([])
    })
})
