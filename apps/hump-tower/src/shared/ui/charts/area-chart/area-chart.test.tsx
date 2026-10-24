import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { act, cleanup, fireEvent, render } from '@testing-library/react'

import { AreaChart } from './index'

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
    { day: 'Tue', ok: 2, failed: 3 },
]

const series = [
    { dataKey: 'ok', label: 'Delivered', variant: 'success' as const },
    { dataKey: 'failed', variant: 'danger' as const },
]

describe('AreaChart', () => {
    test('draws an area and a line per series and opens the tooltip on focus', async () => {
        const { container } = render(<AreaChart data={data} series={series} xAxisKey="day" tooltip legend />)
        const svg = container.querySelector('svg.ts-chart')
        if (!(svg instanceof SVGElement)) throw new Error('chart svg not rendered')

        expect(svg.getAttribute('aria-label')).toBe('Delivered, failed')
        expect(container.querySelector('[data-ts-key=area-ok]')).not.toBeNull()
        expect(container.querySelector('[data-ts-key^=line-failed]')).not.toBeNull()

        await act(async () => {
            svg.focus()
            fireEvent.keyDown(svg, { key: 'ArrowRight' })
        })

        expect(document.body.querySelector('.shadow-xl')?.textContent).toContain('Delivered')
    })

    test('renders a single unstacked point without a tooltip', () => {
        const { container } = render(<AreaChart data={[{ ok: 1 }]} series={[series[0]]} stacked={false} />)

        expect(container.querySelector('[data-ts-key=dot-ok]')).not.toBeNull()
    })
})
