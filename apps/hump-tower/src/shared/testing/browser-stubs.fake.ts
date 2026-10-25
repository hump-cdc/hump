/** Size getters happy-dom leaves at zero. */
const LAYOUT_GETTERS = ['clientWidth', 'clientHeight', 'offsetWidth', 'offsetHeight'] as const

/**
 * Makes every media query report `matches` (`true` puts `useIsMobile` on the mobile layout) until
 * the returned function restores `matchMedia`.
 */
export const stubMatchMedia = (matches: boolean): (() => void) => {
    const original = window.matchMedia
    window.matchMedia = (media: string): MediaQueryList => ({
        matches,
        media,
        onchange: null,
        addEventListener: () => {},
        removeEventListener: () => {},
        addListener: () => {},
        removeListener: () => {},
        dispatchEvent: () => false,
    })

    return () => {
        window.matchMedia = original
    }
}

/**
 * Gives every element a `width` × `height` box, which virtualized lists and measured layouts read,
 * until the returned function restores happy-dom's zero sizes.
 */
export const stubLayout = (width: number, height: number): (() => void) => {
    const originals = LAYOUT_GETTERS.map(
        (key) => [key, Object.getOwnPropertyDescriptor(HTMLElement.prototype, key)] as const,
    )
    const originalRect = Element.prototype.getBoundingClientRect
    for (const key of LAYOUT_GETTERS) {
        Object.defineProperty(HTMLElement.prototype, key, {
            configurable: true,
            get: () => (key.endsWith('Width') ? width : height),
        })
    }
    Element.prototype.getBoundingClientRect = () => new DOMRect(0, 0, width, height)

    return () => {
        for (const [key, descriptor] of originals) {
            if (descriptor) Object.defineProperty(HTMLElement.prototype, key, descriptor)
            else Reflect.deleteProperty(HTMLElement.prototype, key)
        }
        Element.prototype.getBoundingClientRect = originalRect
    }
}

/**
 * Replaces `ResizeObserver` with one that reports every observed element once, right after
 * `observe`, as a browser does on the first layout. Measured layouts (React Flow node sizes, and so
 * its edges) need that first report; happy-dom never sends it. The returned function restores it.
 */
export const stubResizeObserver = (): (() => void) => {
    const original = globalThis.ResizeObserver
    globalThis.ResizeObserver = class {
        readonly #callback: ResizeObserverCallback

        constructor(callback: ResizeObserverCallback) {
            this.#callback = callback
        }

        observe(target: Element): void {
            const entry: ResizeObserverEntry = {
                target,
                contentRect: target.getBoundingClientRect(),
                borderBoxSize: [],
                contentBoxSize: [],
                devicePixelContentBoxSize: [],
            }
            queueMicrotask(() => this.#callback([entry], this))
        }

        unobserve(): void {}

        disconnect(): void {}
    }

    return () => {
        globalThis.ResizeObserver = original
    }
}
