import { describe, expect, test } from 'bun:test'

import { keys, type OperationsFilterKey } from './keys'

const FILTER: OperationsFilterKey = { tables: ['public.users'] }

describe('keys.queries', () => {
    test('builds the metrics key from the duration', () => {
        expect(keys.queries.metrics('5m')).toEqual(['metrics', '5m'])
    })

    test('builds the operations keys from the filter', () => {
        expect(keys.queries.operations(FILTER, 3)).toEqual(['operations', FILTER, 3])
        expect(keys.queries.recentOperations(FILTER)).toEqual(['operations', 'recent', FILTER])
    })

    test('builds the snapshot keys', () => {
        expect(keys.queries.snapshotTables()).toEqual(['snapshot', 'tables'])
        expect(keys.queries.snapshot()).toEqual(['snapshot', 'run'])
        expect(keys.queries.snapshotEstimate('public', 'users', 'id > 1')).toEqual([
            'snapshot',
            'estimate',
            'public',
            'users',
            'id > 1',
        ])
    })
})

describe('keys.prefixes', () => {
    test('prefix every variant of their query', () => {
        expect(keys.queries.metrics('1m').slice(0, keys.prefixes.metrics().length)).toEqual([
            ...keys.prefixes.metrics(),
        ])
        expect(keys.queries.operations(FILTER, 0).slice(0, keys.prefixes.operations().length)).toEqual([
            ...keys.prefixes.operations(),
        ])
        expect(keys.queries.recentOperations(FILTER).slice(0, keys.prefixes.operations().length)).toEqual([
            ...keys.prefixes.operations(),
        ])
    })
})

describe('keys.mutations', () => {
    test('builds the snapshot mutation keys', () => {
        expect(keys.mutations.snapshotStart()).toEqual(['snapshot', 'start'])
        expect(keys.mutations.snapshotStop()).toEqual(['snapshot', 'stop'])
    })
})
