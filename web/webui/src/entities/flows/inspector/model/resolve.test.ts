import { describe, expect, test } from 'bun:test'

import type { MetricsTopology } from '@/entities/metrics/@x/inspector'

import { resolveDelivery, resolveTable, sumTableOps } from './resolve'

const topology: MetricsTopology = {
    tables: [
        {
            name: 'public.users',
            insert: { count: 10, p95: 4 },
            update: { count: 5 },
            delete: { count: 2 },
            truncate: { count: 0 },
        },
    ],
    sinks: [
        {
            type: 'kafka',
            host: 'a:9092',
            operations: 40,
            success: 0.75,
            p95: 12,
            targets: [
                { target: 'users', operations: 30, success: 0.9, p95: 10 },
                { target: 'orders', operations: 10, success: 0.3 },
            ],
        },
        { type: 'kafka', host: 'b:9092', operations: 0, targets: [] },
    ],
    connections: [
        { table: 'public.users', sink: 'kafka', host: 'a:9092', target: 'users', success: 27, failure: 3 },
        { table: 'public.users', sink: 'kafka', host: 'a:9092', target: 'orders', success: 3, failure: 7 },
        { table: 'public.users', sink: 'kafka', host: 'b:9092', target: 'users', success: 0, failure: 0 },
    ],
}

const sink = { type: 'kafka', host: 'a:9092' }

describe('resolveTable', () => {
    test('finds the table and sums its operations', () => {
        const table = resolveTable(topology, 'public.users')
        expect(table?.insert).toEqual({ count: 10, p95: 4 })
        expect(table && sumTableOps(table)).toBe(17)
    })

    test('is undefined for a table the metrics do not report', () => {
        expect(resolveTable(topology, 'public.missing')).toBeUndefined()
    })
})

describe('resolveDelivery', () => {
    test('reports a sink by type and host, deriving its failures', () => {
        expect(resolveDelivery(topology, { kind: 'sink', id: 'kafka|a:9092', name: 'kafka', ...sink })).toEqual({
            operations: 40,
            successRate: 75,
            success: 30,
            failure: 10,
            p95: 12,
        })
    })

    test('leaves rate and failures undefined without an operation', () => {
        const stats = resolveDelivery(topology, {
            kind: 'sink',
            id: 'kafka|b:9092',
            name: 'kafka',
            type: 'kafka',
            host: 'b:9092',
        })
        expect(stats).toEqual({
            operations: 0,
            successRate: undefined,
            success: undefined,
            failure: undefined,
            p95: undefined,
        })
    })

    test('reports a target of its sink', () => {
        const stats = resolveDelivery(topology, {
            kind: 'target',
            id: 'kafka|a:9092/orders',
            name: 'orders',
            sinkLabel: 'kafka',
            ...sink,
        })
        expect(stats).toEqual({ operations: 10, successRate: 30, success: 3, failure: 7, p95: undefined })
    })

    test('sums every target of a summary edge', () => {
        const stats = resolveDelivery(topology, {
            kind: 'edge',
            id: 'e',
            sourceLabel: 'public.users',
            targetLabel: 'kafka',
            table: 'public.users',
            ...sink,
        })
        expect(stats).toEqual({ operations: 40, successRate: 75, success: 30, failure: 10, p95: undefined })
    })

    test('reports only the target of a detailed edge', () => {
        const stats = resolveDelivery(topology, {
            kind: 'edge',
            id: 'e',
            sourceLabel: 'public.users',
            targetLabel: 'kafka/users',
            table: 'public.users',
            target: 'users',
            ...sink,
        })
        expect(stats).toEqual({ operations: 30, successRate: 90, success: 27, failure: 3, p95: undefined })
    })

    test('is undefined for an edge the metrics do not report', () => {
        const stats = resolveDelivery(topology, {
            kind: 'edge',
            id: 'e',
            sourceLabel: 'public.orders',
            targetLabel: 'kafka',
            table: 'public.orders',
            ...sink,
        })
        expect(stats).toBeUndefined()
    })
})
