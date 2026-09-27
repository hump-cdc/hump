import { describe, expect, test } from 'bun:test'

import { type OperationsFilterInput, toOperationsFilter } from './to-request'

describe('toOperationsFilter', () => {
    test('leaves out every facet that allows everything', () => {
        expect(toOperationsFilter(state(), '')).toEqual({ sinks: ['kafka', 'nats'], tables: ['public.users'] })
    })

    test('matches nothing when any facet is empty', () => {
        for (const facet of ['sinks', 'tables', 'operations', 'statuses'] as const) {
            expect(toOperationsFilter(state({ [facet]: [] }), '')).toBeUndefined()
        }
    })

    test('leaves out sinks and tables until config seeded them, rather than matching nothing', () => {
        const filter = toOperationsFilter(
            state({ sinks: [], tables: [], sinksHydrated: false, tablesHydrated: false }),
            '',
        )

        expect(filter).toEqual({})
    })

    test('sends narrowed operations sorted, and statuses in the API vocabulary', () => {
        const filter = toOperationsFilter(state({ operations: ['update', 'delete'], statuses: ['failed'] }), '')

        expect(filter?.operations).toEqual(['delete', 'update'])
        expect(filter?.statuses).toEqual(['failure'])
    })

    test('sorts sinks and tables so the query key is stable', () => {
        const filter = toOperationsFilter(state({ sinks: ['nats', 'kafka'], tables: ['b.t', 'a.t'] }), '')

        expect(filter?.sinks).toEqual(['kafka', 'nats'])
        expect(filter?.tables).toEqual(['a.t', 'b.t'])
    })

    test('trims the search and drops a blank one', () => {
        expect(toOperationsFilter(state(), '  boom ')?.text).toBe('boom')
        expect(toOperationsFilter(state(), '   ')).not.toHaveProperty('text')
    })
})

const state = (overrides: Partial<OperationsFilterInput> = {}): OperationsFilterInput => ({
    sinks: ['nats', 'kafka'],
    tables: ['public.users'],
    operations: ['insert', 'update', 'delete', 'truncate'],
    statuses: ['delivered', 'failed'],
    sinksHydrated: true,
    tablesHydrated: true,
    ...overrides,
})
