import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { act, cleanup, renderHook } from '@testing-library/react'
import type { Edge, Node } from '@xyflow/react'

import { advanceTransition, createTransition, renderTransition, useFlowTransition } from './flow.transition'

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(cleanup)

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

describe('useFlowTransition', () => {
    test('renders the first layout as-is and treats missing input as empty', () => {
        const nodes = [node('a', 0)]
        const { result, rerender } = renderHook(({ input }) => useFlowTransition(input, undefined), {
            initialProps: { input: nodes as Node[] | undefined },
        })
        expect(result.current.nodes).toBe(nodes)
        expect(result.current.edges).toEqual([])

        rerender({ input: undefined })
        expect(result.current.nodes.map(({ id }) => id)).toEqual(['a'])
    })

    test('tweens a layout change frame by frame, then settles on the targets', async () => {
        const target = [node('a', 100)]
        const { result, rerender } = renderHook(({ input }) => useFlowTransition(input, undefined, 40), {
            initialProps: { input: [node('a', 0)] },
        })

        rerender({ input: target })
        expect(result.current.nodes[0].position.y).toBeLessThan(100)

        await act(() => Bun.sleep(120))
        expect(result.current.nodes).toBe(target)
    })

    test('stops animating on unmount', async () => {
        const { rerender, unmount } = renderHook(({ input }) => useFlowTransition(input, undefined, 40), {
            initialProps: { input: [node('a', 0)] },
        })

        rerender({ input: [node('a', 100)] })
        unmount()

        await Bun.sleep(60)
    })
})
