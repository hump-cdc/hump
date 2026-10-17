import { describe, expect, test } from 'bun:test'

import { migrateFlowsFilter } from './filter'

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
