import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { cleanup, render } from '@testing-library/react'

import { ProgressBar } from './index'

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(cleanup)

/** Renders the bar and returns its track and fill elements. */
const renderBar = (value: number) => {
    const view = render(<ProgressBar value={value} size="sm" variant="success" className="x" />)
    const track = view.getByRole('progressbar')
    const fill = track.firstElementChild as HTMLElement

    return { track, fill }
}

describe('ProgressBar', () => {
    test('reports the rounded value and sizes the fill', () => {
        const { track, fill } = renderBar(42.4)

        expect(track.getAttribute('aria-valuenow')).toBe('42')
        expect(track.className).toContain('x')
        expect(fill.style.width).toBe('42.4%')
        expect(fill.className).toContain('bg-success')
    })

    test('clamps values outside 0–100', () => {
        expect(renderBar(150).fill.style.width).toBe('100%')
        cleanup()
        expect(renderBar(-5).fill.style.width).toBe('0%')
    })
})
