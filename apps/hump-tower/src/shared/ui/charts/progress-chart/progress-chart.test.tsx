import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { cleanup, render } from '@testing-library/react'

import { ProgressChart } from './index'

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered)
        GlobalRegistrator.register({ settings: { device: { prefersReducedMotion: 'reduce' } } })
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(cleanup)

/** The green fill bar of a rendered chart. */
const getFill = (container: HTMLElement): HTMLElement | null => container.querySelector('.bg-success')

describe('ProgressChart', () => {
    test.each([
        [30, 'text-danger'],
        [80, 'text-amber'],
        [99, 'text-success'],
    ])('tints the percentage of %d with %s', (value, tint) => {
        const { container } = render(<ProgressChart title="Done" value={value} />)

        expect(container.querySelector('aside')?.className).toContain(tint)
        expect(getFill(container)?.style.width).toBe(`${value}%`)
    })

    test('clamps the fill and shows the red remainder with showError', () => {
        const { container } = render(<ProgressChart title="Done" value={150} showError />)

        expect(getFill(container)?.style.width).toBe('100%')
        expect(container.querySelector('.bg-danger')?.className).not.toContain('opacity-0')
    })

    test('keeps the last width while fading out once the value goes away', () => {
        const { container, rerender } = render(<ProgressChart title="Done" value={40} showError />)
        rerender(<ProgressChart title="Done" showError />)

        expect(container.querySelector('aside')).toBeNull()
        expect(getFill(container)?.style.width).toBe('40%')
        expect(getFill(container)?.className).toContain('opacity-0')
        expect(container.querySelector('.bg-danger')?.className).toContain('opacity-0')
    })

    test('starts empty without a value', () => {
        const { container } = render(<ProgressChart title="Done" />)

        expect(getFill(container)?.style.width).toBe('0%')
    })
})
