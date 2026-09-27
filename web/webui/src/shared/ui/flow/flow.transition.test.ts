import { describe, expect, test } from 'bun:test'

import type { Edge, Node } from '@xyflow/react'

import { advanceTransition, createTransition, renderTransition } from './flow.transition'

const node = (id: string, y: number): Node => ({ id, position: { x: 0, y }, data: {} })
const edge = (source: string, target: string): Edge => ({ id: `${source}>${target}`, source, target })

describe('flow transition', () => {
    const a = node('a', 0)
    const b = node('b', 100)
    const initial = createTransition([a, b], [edge('a', 'b')])

    test('passes new data for the same layout straight through', () => {
        const next = advanceTransition(initial, [{ ...a, data: { value: 1 } }, b], initial.edges, 0, 400)
        expect(renderTransition(next, 1).nodes).toBe(next.nodes)
    })

    test('keeps a removed node fading out while the rest glide', () => {
        const next = advanceTransition(initial, [node('b', 0)], [], 0, 400)
        const { nodes, edges } = renderTransition(next, 0.5)

        expect(nodes.map(({ id, position, style }) => [id, position.y, style?.opacity])).toEqual([
            ['b', 50, 1],
            ['a', 0, 0.5],
        ])
        expect(edges.map(({ style }) => style?.opacity)).toEqual([0.5])
        expect(nodes[1].selectable).toBe(false)
    })

    test('fades an added node in at its position', () => {
        const next = advanceTransition(initial, [a, b, node('c', 200)], initial.edges, 0, 400)
        expect(renderTransition(next, 0.25).nodes[2]).toMatchObject({ position: { y: 200 }, style: { opacity: 0.25 } })
    })

    test('reverses an interrupted fade-out from its current opacity', () => {
        const leaving = advanceTransition(initial, [b], [], 0, 400)
        const back = advanceTransition(leaving, [a, b], [], 200, 400)
        expect(back.from.get('a')?.opacity).toBe(0.5)
        expect(back.leavingNodes).toEqual([])
    })

    test('fades a node re-added after settling in at its new position', () => {
        const left = advanceTransition(initial, [b], [], 0, 400)
        const settled = createTransition(left.nodes, left.edges)
        const back = advanceTransition(settled, [node('a', 300), b], [], 1000, 400)
        expect(back.from.get('a')).toEqual({ x: 0, y: 300, opacity: 0 })
    })
})
