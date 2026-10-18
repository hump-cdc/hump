import { createElement, type ReactElement } from 'react'

import { describe, expect, test } from 'bun:test'

import type { Node } from '@xyflow/react'

import { LABEL_OFFSET_Y } from './flow.consts'
import { collectNodeSpecs, columnLabel, getRevealViewport } from './flow.helpers'
import { TableNode, type TableNodeData } from './node.table'

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
