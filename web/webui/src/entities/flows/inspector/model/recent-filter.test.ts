import { describe, expect, test } from 'bun:test'

import { toRecentFilter } from './recent-filter'

const sink = { type: 'kafka', host: 'a:9092' }

describe('toRecentFilter', () => {
    test('filters a table by its name', () => {
        expect(toRecentFilter({ kind: 'table', id: 'public.users', name: 'public.users' })).toEqual({
            tables: ['public.users'],
        })
    })

    test('filters a sink by type and host', () => {
        expect(toRecentFilter({ kind: 'sink', id: 'kafka|a:9092', name: 'kafka', ...sink })).toEqual({
            sinks: ['kafka'],
            hosts: ['a:9092'],
        })
    })

    test('filters a target by its sink and configured target', () => {
        const filter = toRecentFilter({
            kind: 'target',
            id: 't',
            name: 'users',
            sinkLabel: 'kafka',
            ...sink,
        })
        expect(filter).toEqual({ sinks: ['kafka'], hosts: ['a:9092'], targets: ['users'] })
    })

    test('filters a summary edge by table and sink, a detailed one by its target too', () => {
        const edge = {
            kind: 'edge',
            id: 'e',
            sourceLabel: 's',
            targetLabel: 't',
            table: 'public.users',
            ...sink,
        } as const
        expect(toRecentFilter(edge)).toEqual({ tables: ['public.users'], sinks: ['kafka'], hosts: ['a:9092'] })
        expect(toRecentFilter({ ...edge, target: 'users' })).toEqual({
            tables: ['public.users'],
            sinks: ['kafka'],
            hosts: ['a:9092'],
            targets: ['users'],
        })
    })
})
