import { createElement, type ReactElement } from 'react'

import { afterAll, afterEach, beforeAll, describe, expect, mock, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { act, cleanup, renderHook } from '@testing-library/react'
import type { Edge, Node, NodeChange } from '@xyflow/react'

import { LABEL_OFFSET_Y } from './flow.consts'
import {
    collectNodeSpecs,
    columnLabel,
    getRevealViewport,
    useEdgeAnimation,
    useFlowSelection,
    useMeasuredNodes,
    useSelectionHandlers,
} from './flow.helpers'
import { TableNode, type TableNodeData } from './node.table'

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(cleanup)

const viewport = { x: 0, y: 0, zoom: 1 }
const size = { width: 1000, height: 800 }
const noInsets = { top: 0, right: 0, bottom: 0, left: 0 }
const margin = 10

const ops: TableNodeData['ops'] = {
    insert: { count: 1 },
    update: { count: 2 },
    delete: { count: 3 },
    truncate: { count: 0 },
}

const node = (id: string, y: number): Node => ({ id, position: { x: 0, y }, data: {} })

const table = (id: string, name: string): ReactElement => createElement(TableNode, { key: id, id, name, ops })

describe('columnLabel', () => {
    test('returns nothing for an empty column', () => {
        expect(columnLabel('label', 'TABLES', [], 0)).toEqual([])
    })

    test('places an inert caption above the topmost node', () => {
        expect(columnLabel('label', 'TABLES', [node('a', 50), node('b', -20)], 584)).toEqual([
            {
                id: 'label',
                type: 'label',
                position: { x: 584, y: -20 - LABEL_OFFSET_Y },
                data: { text: 'TABLES' },
                selectable: false,
                focusable: false,
            },
        ])
    })
})

describe('collectNodeSpecs', () => {
    test('resolves config elements to specs in declaration order', () => {
        const { specs, rest } = collectNodeSpecs([table('t1', 'public.users'), table('t2', 'public.orders')])

        expect(specs).toEqual([
            { id: 't1', type: 'table', data: { name: 'public.users', ops } },
            { id: 't2', type: 'table', data: { name: 'public.orders', ops } },
        ])
        expect(rest).toEqual([])
    })

    test('passes everything else through as-is', () => {
        const overlay = createElement('div', { key: 'overlay' })
        const { specs, rest } = collectNodeSpecs([overlay, 'text', table('t1', 'public.users'), null])

        expect(specs.map(({ id }) => id)).toEqual(['t1'])
        expect(rest).toHaveLength(2)
        expect(rest[1]).toBe('text')
    })
})

describe('getRevealViewport', () => {
    test('returns null when the rect is already inside the area', () => {
        expect(
            getRevealViewport({ x: 100, y: 100, width: 200, height: 100 }, viewport, size, noInsets, margin),
        ).toBeNull()
    })

    test('moves the least on each axis to bring an off-screen rect in', () => {
        expect(
            getRevealViewport({ x: -50, y: 750, width: 200, height: 100 }, viewport, size, noInsets, margin),
        ).toEqual({ x: 60, y: -60 })
        expect(
            getRevealViewport({ x: 900, y: -20, width: 200, height: 100 }, viewport, size, noInsets, margin),
        ).toEqual({ x: -110, y: 30 })
    })

    test('keeps the rect clear of the insets', () => {
        const insets = { top: 80, right: 300, bottom: 0, left: 0 }
        expect(getRevealViewport({ x: 600, y: 50, width: 200, height: 100 }, viewport, size, insets, margin)).toEqual({
            x: -110,
            y: 40,
        })
    })

    test('aligns a rect larger than the area to its left/top edge', () => {
        expect(
            getRevealViewport({ x: 100, y: 100, width: 2000, height: 100 }, viewport, size, noInsets, margin),
        ).toEqual({ x: -90, y: 0 })
    })

    test('accounts for the current pan and zoom', () => {
        const zoomed = { x: -100, y: 0, zoom: 0.5 }
        // Screen x = 100 * 0.5 - 100 = -50 → shift right by 60.
        expect(getRevealViewport({ x: 100, y: 100, width: 100, height: 100 }, zoomed, size, noInsets, margin)).toEqual({
            x: -40,
            y: 0,
        })
    })
})

describe('useEdgeAnimation', () => {
    test('flags an edge, keeps it across a repeat call and clears it after the duration', async () => {
        const { result } = renderHook(() => useEdgeAnimation())

        act(() => result.current.animateEdge('e', 20))
        const delay = result.current.animatedEdges.get('e')
        expect(delay).toBeLessThanOrEqual(0)

        act(() => result.current.animateEdge('e', 20))
        expect(result.current.animatedEdges.get('e')).toBe(delay)

        await act(() => Bun.sleep(40))
        expect(result.current.animatedEdges.size).toBe(0)
    })

    test('clears pending timers on unmount', async () => {
        const { result, unmount } = renderHook(() => useEdgeAnimation())

        act(() => result.current.animateEdge('e'))
        unmount()

        expect(result.current.animatedEdges.has('e')).toBe(true)
    })
})

describe('useFlowSelection', () => {
    const nodes: Node[] = [node('a', 0), node('b', 10)]
    const edges: Edge[] = [
        { id: 'e1', source: 'a', target: 'b', style: { strokeWidth: 2 } },
        { id: 'e2', source: 'a', target: 'b' },
    ]

    test('flags the selected item and the animated edges', () => {
        const { result } = renderHook(() => useFlowSelection(nodes, edges, 'e1', new Map([['e1', -120]]), 5))

        expect(result.current.selectedNodes.map(({ selected }) => selected)).toEqual([false, false])
        expect(result.current.selectedEdges).toEqual([
            {
                ...edges[0],
                zIndex: 5,
                style: { strokeWidth: 2, animationDelay: '-120ms' },
                selected: true,
                animated: true,
            },
            { ...edges[1], zIndex: 5, selected: false, animated: false },
        ])
    })

    test('leaves the z-index alone and passes missing edges through', () => {
        const { result } = renderHook(() => useFlowSelection(nodes, undefined, 'a', new Map()))

        expect(result.current.selectedNodes.map(({ selected }) => selected)).toEqual([true, false])
        expect(result.current.selectedEdges).toBeUndefined()

        const withEdges = renderHook(() => useFlowSelection(nodes, edges, undefined, new Map()))
        expect(withEdges.result.current.selectedEdges?.[1]).toEqual({ ...edges[1], selected: false, animated: false })
    })
})

describe('useSelectionHandlers', () => {
    // The handlers ignore the click event, so a bare object stands in for React's synthetic event.
    const event = {} as Parameters<ReturnType<typeof useSelectionHandlers>['handleNodeClick']>[0]
    const tableData: TableNodeData = { name: 'public.users', ops }

    const setup = (selected: string | undefined) => {
        const handlers = {
            onTableNodeSelect: mock(),
            onSinkNodeSelect: mock(),
            onEdgeSelect: mock(),
            onClearSelection: mock(),
        }
        const { result } = renderHook(() => useSelectionHandlers<{ name: string }>({ selected, ...handlers }))
        return { ...handlers, ...result.current }
    }

    test('selects table and sink nodes and ignores the rest', () => {
        const { handleNodeClick, onTableNodeSelect, onSinkNodeSelect, onClearSelection } = setup(undefined)

        handleNodeClick(event, { id: 't', type: 'table', position: { x: 0, y: 0 }, data: tableData })
        handleNodeClick(event, { id: 's', type: 'sink', position: { x: 0, y: 0 }, data: { name: 'kafka' } })
        handleNodeClick(event, { id: 'l', type: 'label', position: { x: 0, y: 0 }, data: {} })

        expect(onTableNodeSelect).toHaveBeenCalledWith('t', tableData)
        expect(onSinkNodeSelect).toHaveBeenCalledWith('s', { name: 'kafka' })
        expect(onClearSelection).not.toHaveBeenCalled()
    })

    test('selects an edge with its endpoints', () => {
        const { handleEdgeClick, onEdgeSelect } = setup(undefined)

        handleEdgeClick(event, { id: 'e', source: 'a', target: 'b', targetHandle: 'h' })

        expect(onEdgeSelect).toHaveBeenCalledWith('e', { source: 'a', target: 'b', targetHandle: 'h' })
    })

    test('clears the selection on a re-click or a pane click', () => {
        const { handleNodeClick, handleEdgeClick, handlePaneClick, onClearSelection, onTableNodeSelect } = setup('x')

        handleNodeClick(event, { id: 'x', type: 'table', position: { x: 0, y: 0 }, data: tableData })
        handleEdgeClick(event, { id: 'x', source: 'a', target: 'b' })
        handlePaneClick()

        expect(onClearSelection).toHaveBeenCalledTimes(3)
        expect(onTableNodeSelect).not.toHaveBeenCalled()
    })
})

describe('useMeasuredNodes', () => {
    test('carries recorded sizes onto rebuilt nodes and forwards every change', () => {
        const onNodesChange = mock()
        const measured: Node = { ...node('b', 0), measured: { width: 1, height: 1 } }
        const { result, rerender } = renderHook(({ nodes }) => useMeasuredNodes(nodes, onNodesChange), {
            initialProps: { nodes: [node('a', 0), measured] },
        })
        const changes: NodeChange[] = [
            { id: 'a', type: 'dimensions', dimensions: { width: 200, height: 100 } },
            { id: 'b', type: 'select', selected: true },
        ]

        act(() => result.current.handleNodesChange(changes))
        act(() => result.current.handleNodesChange([{ id: 'b', type: 'select', selected: false }]))
        rerender({ nodes: [node('a', 0), measured] })

        expect(onNodesChange).toHaveBeenCalledTimes(2)
        expect(onNodesChange).toHaveBeenCalledWith(changes)
        expect(result.current.measuredNodes?.[0].measured).toEqual({ width: 200, height: 100 })
        expect(result.current.measuredNodes?.[1]).toBe(measured)
    })
})
