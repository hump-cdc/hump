import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { act, cleanup, renderHook } from '@testing-library/react'

import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'

import { useElementSize } from './use-element-size'

/** Observers created through the stubbed `ResizeObserver`, so a test can fire their callbacks. */
const observers: FakeResizeObserver[] = []

/** A `ResizeObserver` that reports only when the test calls `resize`. */
class FakeResizeObserver {
    disconnected = false

    constructor(private readonly callback: () => void) {
        observers.push(this)
    }

    observe(): void {}

    disconnect(): void {
        this.disconnected = true
    }

    resize(): void {
        this.callback()
    }
}

let originalResizeObserver: typeof ResizeObserver

beforeAll(() => {
    registerDom()
    originalResizeObserver = globalThis.ResizeObserver
    Object.defineProperty(globalThis, 'ResizeObserver', { value: FakeResizeObserver, configurable: true })
})
afterAll(async () => {
    Object.defineProperty(globalThis, 'ResizeObserver', { value: originalResizeObserver, configurable: true })
    await unregisterDom()
})
afterEach(() => {
    cleanup()
    observers.length = 0
})

describe('useElementSize', () => {
    test('reports zero until an element is attached', () => {
        const { result } = renderHook(() => useElementSize<HTMLDivElement>())

        expect(result.current[1]).toEqual({ width: 0, height: 0 })
    })

    test('measures the attached element and follows its resizes', () => {
        const element = sizedElement(120, 40)
        const { result } = renderHook(() => useElementSize<HTMLDivElement>())

        act(() => result.current[0](element))
        expect(result.current[1]).toEqual({ width: 120, height: 40 })

        const unchanged = result.current[1]
        act(() => observers[0].resize())
        expect(result.current[1]).toBe(unchanged)

        setSize(element, 200, 50)
        act(() => observers[0].resize())
        expect(result.current[1]).toEqual({ width: 200, height: 50 })
    })

    test('drops back to zero and disconnects once the element detaches', () => {
        const { result } = renderHook(() => useElementSize<HTMLDivElement>())

        act(() => result.current[0](sizedElement(10, 10)))
        act(() => result.current[0](null))

        expect(result.current[1]).toEqual({ width: 0, height: 0 })
        expect(observers[0].disconnected).toBe(true)
    })
})

/** A div whose `offsetWidth`/`offsetHeight` report the given size. */
const sizedElement = (width: number, height: number): HTMLDivElement => {
    const element = document.createElement('div')
    setSize(element, width, height)
    return element
}

/** Overrides the layout size happy-dom reports for `element`. */
const setSize = (element: HTMLElement, width: number, height: number): void => {
    Object.defineProperty(element, 'offsetWidth', { value: width, configurable: true })
    Object.defineProperty(element, 'offsetHeight', { value: height, configurable: true })
}
