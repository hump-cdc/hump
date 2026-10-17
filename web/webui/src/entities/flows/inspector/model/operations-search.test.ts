import { describe, expect, test } from 'bun:test'

import { toOperationsSearch } from './operations-search'

const sink = { type: 'kafka', host: 'a:9092' }

describe('toOperationsSearch', () => {
    test('targets a table by its name', () => {
        expect(toOperationsSearch({ kind: 'table', id: 'public.users', name: 'public.users' })).toEqual({
            table: 'public.users',
        })
    })

    test('targets a sink by its type', () => {
        expect(toOperationsSearch({ kind: 'sink', id: 'kafka|a:9092', name: 'kafka', ...sink })).toEqual({
            sink: 'kafka',
        })
    })

    test('targets a sink target by its sink and target', () => {
        expect(toOperationsSearch({ kind: 'target', id: 't', name: 'users', sinkLabel: 'kafka', ...sink })).toEqual({
            sink: 'kafka',
            target: 'users',
        })
    })

    test('targets a detailed edge by table, sink and target', () => {
        const edge = {
            kind: 'edge',
            id: 'e',
            sourceLabel: 's',
            targetLabel: 't',
            table: 'public.users',
            ...sink,
        } as const
        expect(toOperationsSearch({ ...edge, target: 'users' })).toEqual({
            table: 'public.users',
            sink: 'kafka',
            target: 'users',
        })
        expect(toOperationsSearch(edge)).toEqual({ table: 'public.users', sink: 'kafka' })
    })
})
