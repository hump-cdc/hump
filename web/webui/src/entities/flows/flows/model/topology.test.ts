import { describe, expect, test } from 'bun:test'

import type { MetricsTopology } from '@/entities/metrics/@x/flows'

import { sinkId, tableId, toFlowsTopology } from './topology'

const topology: MetricsTopology = {
    tables: [{ name: 'public.events', insert: 1_240_000, update: 0, delete: 3, truncate: 0 }],
    sinks: [
        {
            type: 'http',
            host: 'localhost:5050',
            operations: 30,
            success: 0.4,
            targets: [
                { target: 'http://localhost:5050/a', operations: 10, success: 1, p95: 12 },
                { target: 'http://localhost:5050/b', operations: 20 },
            ],
        },
    ],
    connections: [
        {
            table: 'public.events',
            sink: 'http',
            host: 'localhost:5050',
            target: 'http://localhost:5050/a',
            operations: 10,
        },
        {
            table: 'public.events',
            sink: 'http',
            host: 'localhost:5050',
            target: 'http://localhost:5050/b',
            operations: 20,
        },
    ],
}

describe('toFlowsTopology', () => {
    const flows = toFlowsTopology(topology)
    const sink = sinkId('http', 'localhost:5050')

    test('ids match the fields of a live operation', () => {
        expect(flows.tables[0].id).toBe(tableId('public', 'events'))
        expect(flows.sinks[0].id).toBe(sink)
        expect(flows.detailedEdges.map(({ source, target, targetHandle }) => [source, target, targetHandle])).toEqual([
            ['public.events', sink, 'http://localhost:5050/a'],
            ['public.events', sink, 'http://localhost:5050/b'],
        ])
    })

    test('sums connections into one summary edge per table and sink', () => {
        expect(flows.edges).toHaveLength(1)
        expect(flows.edges[0]).toMatchObject({
            source: 'public.events',
            target: sink,
            style: { opacity: 1, strokeWidth: 5 },
        })
    })

    test('weights detailed edges by their share of the busiest one', () => {
        expect(flows.detailedEdges.map(({ style }) => style?.strokeWidth)).toEqual([3.25, 5])
    })

    test('passes sink and target figures through, leaving what is missing undefined', () => {
        expect(flows.tables[0].ops.insert.count).toBe(1240000)
        expect(flows.sinks[0]).toMatchObject({ name: 'http', p95: undefined, successRate: 40, health: 'error' })
        expect(flows.detailedSinks[0].targets).toEqual([
            expect.objectContaining({ kind: 'url', p95: 12, successRate: 100, health: 'ok' }),
            expect.objectContaining({ kind: 'url', p95: undefined, successRate: undefined, health: 'ok' }),
        ])
    })
})
