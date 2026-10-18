import { afterEach, beforeEach, describe, expect, type Mock, spyOn, test } from 'bun:test'

import type { MetricsTopology } from '@/entities/metrics/@x/flows'

import { filterFlowsTopology, migrateFlowsFilter, useFlowsFilterStore } from './filter'
import { sinkId, toFlowsTopology } from './topology'

const KAFKA = sinkId('kafka', 'localhost:9092')
const HTTP = sinkId('http', 'localhost:5050')

/** Two tables, each delivered to both sinks. */
const METRICS: MetricsTopology = {
    tables: ['public.users', 'public.orders'].map((name) => ({
        name,
        insert: { count: 1 },
        update: { count: 0 },
        delete: { count: 0 },
        truncate: { count: 0 },
    })),
    sinks: [
        { type: 'kafka', host: 'localhost:9092', operations: 2, targets: [{ target: 'users', operations: 2 }] },
        { type: 'http', host: 'localhost:5050', operations: 2, targets: [{ target: '/hook', operations: 2 }] },
    ],
    connections: ['public.users', 'public.orders'].flatMap((table) => [
        { table, sink: 'kafka', host: 'localhost:9092', target: 'users', success: 1, failure: 0 },
        { table, sink: 'http', host: 'localhost:5050', target: '/hook', success: 1, failure: 0 },
    ]),
}

const TOPOLOGY = toFlowsTopology(METRICS)

describe('migrateFlowsFilter', () => {
    test('keeps stored selections', () => {
        expect(migrateFlowsFilter({ tables: ['public.users'], sinks: 'all' })).toEqual({
            tables: ['public.users'],
            sinks: 'all',
        })
    })

    test('turns a facet that is not a list of ids into `all`', () => {
        expect(migrateFlowsFilter({ tables: [1, 2], sinks: 'kafka' })).toEqual({ tables: 'all', sinks: 'all' })
    })

    test('falls back to `all` when nothing usable was stored', () => {
        expect(migrateFlowsFilter(null)).toEqual({ tables: 'all', sinks: 'all' })
        expect(migrateFlowsFilter({})).toEqual({ tables: 'all', sinks: 'all' })
    })
})

describe('filterFlowsTopology', () => {
    test('returns the topology itself when both facets are `all`', () => {
        expect(filterFlowsTopology(TOPOLOGY, 'all', 'all')).toBe(TOPOLOGY)
    })

    test('drops unselected tables together with their edges', () => {
        const filtered = filterFlowsTopology(TOPOLOGY, ['public.users'], 'all')

        expect(filtered.tables.map(({ id }) => id)).toEqual(['public.users'])
        expect(filtered.sinks.map(({ id }) => id)).toEqual([KAFKA, HTTP])
        expect(filtered.detailedSinks.map(({ id }) => id)).toEqual([KAFKA, HTTP])
        expect(filtered.edges.map(({ source, target }) => [source, target])).toEqual([
            ['public.users', KAFKA],
            ['public.users', HTTP],
        ])
        expect(filtered.detailedEdges.map(({ source }) => source)).toEqual(['public.users', 'public.users'])
    })

    test('drops unselected sinks from both views together with their edges', () => {
        const filtered = filterFlowsTopology(TOPOLOGY, 'all', [HTTP])

        expect(filtered.tables.map(({ id }) => id)).toEqual(['public.users', 'public.orders'])
        expect(filtered.sinks.map(({ id }) => id)).toEqual([HTTP])
        expect(filtered.detailedSinks.map(({ id }) => id)).toEqual([HTTP])
        expect(filtered.edges.map(({ source, target }) => [source, target])).toEqual([
            ['public.users', HTTP],
            ['public.orders', HTTP],
        ])
        expect(filtered.detailedEdges.map(({ target }) => target)).toEqual([HTTP, HTTP])
    })

    test('keeps only the edges between a selected table and a selected sink', () => {
        const filtered = filterFlowsTopology(TOPOLOGY, ['public.orders'], [KAFKA])

        expect(filtered.edges.map(({ source, target }) => [source, target])).toEqual([['public.orders', KAFKA]])
        expect(filtered.detailedEdges.map(({ source, target }) => [source, target])).toEqual([['public.orders', KAFKA]])
    })

    test('an empty selection hides everything it filters', () => {
        const filtered = filterFlowsTopology(TOPOLOGY, [], 'all')

        expect(filtered.tables).toEqual([])
        expect(filtered.edges).toEqual([])
        expect(filtered.detailedEdges).toEqual([])
        expect(filtered.sinks).toHaveLength(2)
    })
})

describe('useFlowsFilterStore', () => {
    // Bun has no `localStorage`, so the persist middleware warns on every `set`; the warning is muted.
    let warn: Mock<typeof console.warn>

    beforeEach(() => {
        warn = spyOn(console, 'warn').mockImplementation(() => {})
        useFlowsFilterStore.setState(useFlowsFilterStore.getInitialState(), true)
    })

    afterEach(() => {
        warn.mockRestore()
    })

    test('starts with every table and sink selected', () => {
        expect(useFlowsFilterStore.getState()).toMatchObject({ tables: 'all', sinks: 'all' })
    })

    test('setTables and setSinks overwrite their own facet only', () => {
        useFlowsFilterStore.getState().setTables(['public.users'])
        expect(useFlowsFilterStore.getState()).toMatchObject({ tables: ['public.users'], sinks: 'all' })

        useFlowsFilterStore.getState().setSinks([KAFKA])
        expect(useFlowsFilterStore.getState()).toMatchObject({ tables: ['public.users'], sinks: [KAFKA] })
    })
})
