import { describe, expect, test } from 'bun:test'

import { getRevealViewport } from './flow.helpers'

const viewport = { x: 0, y: 0, zoom: 1 }
const size = { width: 1000, height: 800 }
const noInsets = { top: 0, right: 0, bottom: 0, left: 0 }
const margin = 10

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
