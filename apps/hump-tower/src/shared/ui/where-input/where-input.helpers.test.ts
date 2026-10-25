import { afterAll, beforeAll, describe, expect, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import type { PrismEditor } from 'prism-react-editor'

import { caretAnchor, clampHeight, collapsedPreview, contentHeightOf } from './where-input.helpers'

/** Caret position as the cursor extension reports it. */
type Caret = { left: number; top: number; height: number }

/** A line element at the given offsets, with a bounding box at (`left`, `top`). */
const line = (offsetTop: number, offsetHeight: number, left = 0, top = 0): HTMLElement => {
    const element = document.createElement('div')
    Object.defineProperties(element, { offsetTop: { value: offsetTop }, offsetHeight: { value: offsetHeight } })
    element.getBoundingClientRect = () => new DOMRect(left, top, 100, offsetHeight)
    return element
}

/** The slice of `PrismEditor` the helpers read: its line elements and the cursor extension. */
const editorWith = (lines: HTMLElement[] | undefined, caret?: Caret): PrismEditor =>
    ({
        lines,
        extensions: { cursor: caret && { getPosition: () => caret } },
    }) as unknown as PrismEditor

// `caretAnchor` builds DOMRects and the fake lines are DOM elements.
beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})

describe('clampHeight', () => {
    test('keeps room for the minimum rows', () => {
        expect(clampHeight(0)).toBe(76)
        expect(clampHeight(40)).toBe(76)
    })

    test('follows the content between the minimum and maximum rows', () => {
        expect(clampHeight(100)).toBe(102)
    })

    test('stops growing at the maximum rows', () => {
        expect(clampHeight(500)).toBe(216)
    })
})

describe('collapsedPreview', () => {
    test('is null for a blank value', () => {
        expect(collapsedPreview('')).toBeNull()
        expect(collapsedPreview(' \n\t\n ')).toBeNull()
    })

    test('highlights the first non-blank line', () => {
        const preview = collapsedPreview("\n  \nstatus = 'active'\n\n")

        expect(preview?.more).toBe(false)
        expect(preview?.html).toStartWith('status ')
        expect(preview?.html).toContain('<span class="token string">\'active\'</span>')
    })

    test('flags non-blank lines after the first one', () => {
        expect(collapsedPreview('id > 1\n  and status is null')).toMatchObject({
            html: expect.stringContaining('id'),
            more: true,
        })
    })
})

describe('contentHeightOf', () => {
    test('is zero before the editor has lines', () => {
        expect(contentHeightOf(editorWith(undefined))).toBe(0)
    })

    test('measures to the bottom of the last line plus padding', () => {
        expect(contentHeightOf(editorWith([line(0, 0), line(7, 20), line(27, 40)]))).toBe(81)
    })
})

describe('caretAnchor', () => {
    test('is an empty rect without lines or a caret', () => {
        expect(caretAnchor(editorWith(undefined)).getBoundingClientRect()).toMatchObject({ x: 0, y: 0, height: 0 })
        expect(caretAnchor(editorWith([line(0, 20)])).getBoundingClientRect()).toMatchObject({ x: 0, height: 0 })
    })

    test('offsets the caret by the overlay line box', () => {
        const rect = caretAnchor(
            editorWith([line(0, 20, 10, 50)], { left: 5, top: 3, height: 18 }),
        ).getBoundingClientRect()

        expect(rect).toMatchObject({ x: 15, y: 53, width: 0, height: 18 })
    })
})
