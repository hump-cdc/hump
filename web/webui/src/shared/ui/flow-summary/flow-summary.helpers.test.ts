import { createElement, type ReactElement } from 'react'

import { describe, expect, test } from 'bun:test'

import { TableNode, type TableNodeData } from '@/shared/ui/flow'

import { partitionChildren } from './flow-summary.helpers'
import { SinkNode } from './node.sink'

const ops: TableNodeData['ops'] = {
    insert: { count: 0 },
    update: { count: 0 },
    delete: { count: 0 },
    truncate: { count: 0 },
}

const table = (id: string): ReactElement => createElement(TableNode, { key: id, id, name: id, ops })

const sink = (id: string): ReactElement =>
    createElement(SinkNode, { key: id, id, name: id, type: 'kafka', operations: 0, health: 'ok' })

describe('partitionChildren', () => {
    test('centres tables and sinks in their columns and captions each column', () => {
        const { nodes, rest } = partitionChildren([table('t1'), sink('s1'), table('t2'), table('t3'), sink('s2')])

        expect(nodes.map(({ id, type, position }) => [id, type, position.x, position.y])).toEqual([
            ['t1', 'table', 0, -136],
            ['t2', 'table', 0, 0],
            ['t3', 'table', 0, 136],
            ['s1', 'sink', 584, -95],
            ['s2', 'sink', 584, 95],
            ['label-tables', 'label', 0, -172],
            ['label-sinks', 'label', 584, -131],
        ])
        expect(nodes[0].data).toEqual({ name: 't1', ops })
        expect(rest).toEqual([])
    })

    test('leaves out the caption of an empty column and passes overlays through', () => {
        const overlay = createElement('div', { key: 'overlay' })
        const { nodes, rest } = partitionChildren([table('t1'), overlay])

        expect(nodes.map(({ id, position }) => [id, position.y])).toEqual([
            ['t1', 0],
            ['label-tables', -36],
        ])
        expect(rest).toHaveLength(1)
    })

    test('returns no nodes without config children', () => {
        expect(partitionChildren(null)).toEqual({ nodes: [], rest: [] })
    })
})
