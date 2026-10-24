import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { cleanup, render } from '@testing-library/react'

import { NumberFlow } from './index'

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(cleanup)

/** Text NumberFlow exposes for a value: its accessible label, or the rendered text. */
const textOf = (element: Element): string => element.getAttribute('aria-label') ?? element.textContent ?? ''

describe('NumberFlow', () => {
    test('renders a dash while the value is undefined', () => {
        const view = render(<NumberFlow value={undefined} className="x" />)

        const dash = view.getByText('—')
        expect(dash.className).toContain('x')
    })

    test.each([
        ['integer', 12345, undefined],
        ['decimal', 1.234, undefined],
        ['compact', 1500, undefined],
        ['compact', 12.5, undefined],
        ['percent', 42, undefined],
        ['bytes', 2048, '/s'],
        ['latency', 0.25, undefined],
    ] as const)('renders a %s value', (kind, value, suffix) => {
        const view = render(<NumberFlow value={value} kind={kind} suffix={suffix} className="x" />)

        const flow = view.container.firstElementChild as Element
        expect(flow).not.toBeNull()
        expect(flow.getAttribute('class')).toContain('font-mono')
        if (suffix) expect(textOf(flow)).toContain(suffix)
    })

    test.each([
        [3725, 3],
        [125, 2],
        [42, 1],
    ])('renders a %d-second duration in units', (seconds, units) => {
        const view = render(<NumberFlow value={seconds} kind="duration" suffix=" ago" />)

        const group = view.container.firstElementChild as Element
        expect(group.className).toContain('font-mono')
        expect(group.children).toHaveLength(units)
    })
})
