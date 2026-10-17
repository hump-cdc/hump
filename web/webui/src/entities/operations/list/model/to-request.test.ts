import { describe, expect, test } from 'bun:test'

import { type OperationsFilterInput, toOperationsFilter } from './to-request'

describe('toOperationsFilter', () => {
    test('leaves out every facet that allows everything', () => {
        expect(toOperationsFilter(state(), '')).toEqual({})
    })

    test('matches nothing when any facet is empty', () => {
        for (const facet of ['sinks', 'targets', 'tables', 'operations', 'statuses'] as const) {
            expect(toOperationsFilter(state({ [facet]: [] }), '')).toBeUndefined()
        }
    })

    test('sends narrowed operations sorted, and statuses in the API vocabulary', () => {
        const filter = toOperationsFilter(state({ operations: ['update', 'delete'], statuses: ['failed'] }), '')

        expect(filter?.operations).toEqual(['delete', 'update'])
        expect(filter?.statuses).toEqual(['failure'])
    })

    test('sorts sinks, targets and tables so the query key is stable', () => {
        const filter = toOperationsFilter(
            state({ sinks: ['nats', 'kafka'], targets: ['users', 'orders'], tables: ['b.t', 'a.t'] }),
            '',
        )

        expect(filter?.sinks).toEqual(['kafka', 'nats'])
        expect(filter?.targets).toEqual(['orders', 'users'])
        expect(filter?.tables).toEqual(['a.t', 'b.t'])
    })

    test('trims the search and drops a blank one', () => {
        expect(toOperationsFilter(state(), '  boom ')?.text).toBe('boom')
        expect(toOperationsFilter(state(), '   ')).not.toHaveProperty('text')
    })
})

const state = (overrides: Partial<OperationsFilterInput> = {}): OperationsFilterInput => ({
    sinks: null,
    targets: null,
    tables: null,
    operations: ['insert', 'update', 'delete', 'truncate'],
    statuses: ['delivered', 'failed'],
    ...overrides,
})
