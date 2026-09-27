import { describe, expect, test } from 'bun:test'

import { defaultFilterValues } from '@/entities/operations/filter'

import { operationsSearchSchema, toFilterValues } from './search'

describe('operationsSearchSchema', () => {
    test('reads single values, repeated keys and numbers as string lists', () => {
        expect(operationsSearchSchema.parse({ table: 'public.users', sink: ['kafka', 'nats'], target: 123 })).toEqual({
            table: ['public.users'],
            sink: ['kafka', 'nats'],
            target: ['123'],
        })
    })

    test('drops a malformed facet', () => {
        expect(operationsSearchSchema.parse({ table: { a: 1 } })).toEqual({})
    })
})

describe('toFilterValues', () => {
    test('maps each param to its facet', () => {
        expect(
            toFilterValues({
                table: ['public.users'],
                sink: ['kafka'],
                target: ['users'],
                operation: ['insert'],
                result: ['failed'],
            }),
        ).toEqual({
            tables: ['public.users'],
            sinks: ['kafka'],
            targets: ['users'],
            operations: ['insert'],
            statuses: ['failed'],
            search: '',
        })
    })

    test('resets the facets left out to all', () => {
        expect(toFilterValues({ table: ['public.users'] })).toEqual({
            ...defaultFilterValues,
            tables: ['public.users'],
        })
    })

    test('drops unknown operations and results', () => {
        expect(toFilterValues({ operation: ['insert', 'upsert'], result: ['delivered', 'success'] })).toEqual({
            ...defaultFilterValues,
            operations: ['insert'],
            statuses: ['delivered'],
        })
    })

    test('treats a facet with no known value as left out', () => {
        expect(toFilterValues({ table: ['public.users'], result: ['success'] })).toEqual({
            ...defaultFilterValues,
            tables: ['public.users'],
        })
        expect(toFilterValues({ result: ['success'] })).toBeUndefined()
    })

    test('is undefined without params', () => {
        expect(toFilterValues({})).toBeUndefined()
    })
})
