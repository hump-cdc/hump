import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { act, cleanup, fireEvent, render } from '@testing-library/react'

import { BarChart } from './index'

// Reduced motion keeps the motion renderer and NumberFlow from starting animations that reject on unmount.
beforeAll(() => {
    if (!GlobalRegistrator.isRegistered)
        GlobalRegistrator.register({ settings: { device: { prefersReducedMotion: 'reduce' } } })
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(cleanup)

const data = [
    { day: 'Mon', ok: 4, failed: 1 },
    { day: 'Tue', ok: 0, failed: 0 },
]

/** Moves keyboard focus onto the first x step so the tooltip opens. */
const focusFirstStep = async (container: HTMLElement): Promise<void> => {
    const svg = container.querySelector('svg.ts-chart')
    if (!(svg instanceof SVGElement)) throw new Error('chart svg not rendered')

    await act(async () => {
        svg.focus()
        fireEvent.keyDown(svg, { key: 'ArrowRight' })
    })
}

describe('BarChart', () => {
    test('stacks the series, draws stand-in bars for empty steps and opens the tooltip on focus', async () => {
        const { container } = render(
            <BarChart
                data={data}
                series={[
                    { dataKey: 'ok', label: 'Delivered', variant: 'success' },
                    { dataKey: 'failed', variant: 'danger' },
                ]}
                xAxisKey="day"
                tooltip
                tooltipValue={{ suffix: ' ops' }}
                legend
            />,
        )

        expect(container.querySelector('svg')?.getAttribute('aria-label')).toBe('Delivered, failed')
        expect(container.querySelectorAll('[data-ts-key=baseline] rect')).toHaveLength(1)

        await focusFirstStep(container)

        expect(document.body.querySelector('.shadow-xl')?.textContent).toContain('Delivered')
    })

    test('groups the series side by side when not stacked', () => {
        const { container } = render(
            <BarChart data={data} series={[{ dataKey: 'ok', variant: 'primary' }]} stacked={false} />,
        )

        expect(container.querySelectorAll('[data-ts-key=bars] path')).toHaveLength(2)
    })

    test('falls back to a generic label and baseline color without series', () => {
        const { container } = render(<BarChart data={data} series={[]} tooltip />)

        expect(container.querySelector('svg')?.getAttribute('aria-label')).toBe('Bar chart')
        expect(container.querySelector('[data-ts-key=baseline] rect')?.getAttribute('fill')).toBe(
            'var(--hairline-strong)',
        )
    })
})
