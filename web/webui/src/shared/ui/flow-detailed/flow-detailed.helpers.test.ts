import { createElement, type ReactElement } from 'react'

import { describe, expect, test } from 'bun:test'

import type { Node } from '@xyflow/react'

import { TableNode, type TableNodeData } from '@/shared/ui/flow'

import { getRevealId, partitionChildren } from './flow-detailed.helpers'
import { GhostNode } from './node.ghost'
import { SinkNode, type SinkTarget } from './node.sink'

const ops: TableNodeData['ops'] = {
    insert: { count: 0 },
    update: { count: 0 },
    delete: { count: 0 },
    truncate: { count: 0 },
}

const target = (id: string): SinkTarget => ({ id, name: id, kind: 'topic', count: 0, health: 'ok' })

const table = (id: string): ReactElement => createElement(TableNode, { key: id, id, name: id, ops })

const sink = (id: string, targetIds: string[]): ReactElement =>
    createElement(SinkNode, { key: id, id, name: id, type: 'kafka', count: 0, targets: targetIds.map(target) })

const ghost = (id: string): ReactElement => createElement(GhostNode, { key: id, id, type: 'clickhouse' })

const positions = (nodes: Node[]): [string, number, number][] =>
    nodes.map(({ id, position }) => [id, position.x, position.y])

describe('partitionChildren', () => {
    test('stacks tables, then sinks and ghosts by their estimated heights, centred on y=0', () => {
        const { nodes } = partitionChildren([
            ghost('g1'),
            table('t1'),
            sink('s1', ['a', 'b']),
            ghost('g2'),
            table('t2'),
            sink('s2', []),
        ])

        expect(positions(nodes)).toEqual([
            ['t1', 0, -120],
            ['t2', 0, 16],
            ['s1', 584, -186],
            ['s2', 584, -33],
            ['g1', 584, 56],
            ['g2', 584, 132],
            ['label-tables', 0, -156],
            ['label-sinks', 584, -222],
        ])
    })

    test('makes ghosts unselectable and keeps sink data', () => {
        const { nodes } = partitionChildren([sink('s1', ['a']), ghost('g1')])

        expect(nodes.find(({ id }) => id === 'g1')).toMatchObject({ type: 'ghost', selectable: false })
        expect(nodes.find(({ id }) => id === 's1')).not.toHaveProperty('selectable')
        expect(nodes.find(({ id }) => id === 's1')?.data).toMatchObject({ targets: [target('a')] })
    })

    test('spaces a ghost-only column by the ghost gap', () => {
        const { nodes } = partitionChildren([ghost('g1'), ghost('g2')])

        expect(positions(nodes)).toEqual([
            ['g1', 584, -65],
            ['g2', 584, 11],
            ['label-sinks', 584, -101],
        ])
    })

    test('passes overlays through', () => {
        const overlay = createElement('div', { key: 'overlay' })
        const { nodes, rest } = partitionChildren([overlay])

        expect(nodes).toEqual([])
        expect(rest).toHaveLength(1)
    })
})

describe('getRevealId', () => {
    const nodes: Node[] = [
        { id: 't1', type: 'table', position: { x: 0, y: 0 }, data: {} },
        { id: 's1', type: 'sink', position: { x: 0, y: 0 }, data: { targets: [target('orders')] } },
    ]

    test('resolves a selected target row to its sink', () => {
        expect(getRevealId(nodes, 's1:orders')).toBe('s1')
    })

    test('passes a node or edge id through', () => {
        expect(getRevealId(nodes, 't1')).toBe('t1')
        expect(getRevealId(nodes, 't1>s1')).toBe('t1>s1')
        expect(getRevealId(nodes, undefined)).toBeUndefined()
    })
})
