import type { ReactNode } from 'react'

import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { cleanup, render } from '@testing-library/react'

import { TooltipProvider } from '@/shared/ui/tooltip'

import { AreaChartCard, BarChartCard, Card, CardContent, CardFooter, CardHeader, InfoCard, MetricCard } from './index'

// Reduced motion keeps the chart renderer and NumberFlow from starting animations that reject on unmount.
beforeAll(() => {
    if (!GlobalRegistrator.isRegistered)
        GlobalRegistrator.register({ settings: { device: { prefersReducedMotion: 'reduce' } } })
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(cleanup)

/** Renders under the tooltip provider the hint icons need. */
const renderWithTooltips = (node: ReactNode) => render(<TooltipProvider>{node}</TooltipProvider>)

const values = (...numbers: number[]) => numbers.map((value, index) => ({ date: `d${index}`, value }))

describe('Card', () => {
    test('separates the header and footer with borders by default', () => {
        const { container } = render(
            <Card>
                <CardHeader aside={<b>aside</b>}>Head</CardHeader>
                <CardContent>Body</CardContent>
                <CardFooter aside={<b>more</b>}>Foot</CardFooter>
            </Card>,
        )
        const [header, content, footer] = container.firstElementChild?.children ?? []

        expect(container.firstElementChild?.className).toContain('p-0')
        expect(header.className).toContain('border-b')
        expect(header.textContent).toBe('Headaside')
        expect(content.className).toContain('flex-1')
        expect(footer.className).toContain('border-t')
        expect(footer.textContent).toBe('Footmore')
    })

    test('drops the borders when not separated', () => {
        const { container } = render(
            <Card separated={false}>
                <CardHeader>Head</CardHeader>
                <CardFooter>Foot</CardFooter>
            </Card>,
        )
        const [header, footer] = container.firstElementChild?.children ?? []

        expect(container.firstElementChild?.className).toContain('p-4')
        expect(header.className).not.toContain('border-b')
        expect(header.children).toHaveLength(0)
        expect(footer.className).not.toContain('border-t')
    })

    test('falls back to separated for a null variant and merges onto the child with asChild', () => {
        const { container } = render(
            <Card asChild separated={null}>
                <section>Section</section>
            </Card>,
        )

        expect(container.firstElementChild?.tagName).toBe('SECTION')
        expect(container.firstElementChild?.className).toContain('p-0')
    })
})

describe('InfoCard', () => {
    test('renders the title, tinted content and footer', () => {
        const { container } = render(
            <InfoCard title="Sink" variant="failed" footer="since 1m">
                kafka
            </InfoCard>,
        )

        expect(container.textContent).toBe('Sinkkafkasince 1m')
        expect(container.querySelector('.text-danger')?.textContent).toBe('kafka')
    })

    test('omits the footer when absent', () => {
        const { container } = render(<InfoCard title="Sink">kafka</InfoCard>)

        expect(container.querySelector('.text-xs')).toBeNull()
    })
})

describe('MetricCard', () => {
    test('renders a separated card with hint, value, chart, footer and an upward good trend', () => {
        const { container } = renderWithTooltips(
            <MetricCard
                title="Rate"
                hint="per second"
                value="12"
                values={values(1, 2)}
                chartSize="lg"
                separated
                footer="ok"
            />,
        )

        expect(container.textContent).toContain('Rate')
        expect(container.textContent).toContain('12')
        expect(container.querySelector('.h-32')).not.toBeNull()
        expect(container.querySelector('svg.text-success')).not.toBeNull()
        expect(container.textContent).toContain('ok')
    })

    test('marks a drop as bad for a positive trend and a rise as bad for a negative one', () => {
        const { container: drop } = render(<MetricCard title="Rate" values={values(2, 1)} chartSize="md" />)
        const { container: rise } = render(
            <MetricCard title="Errors" values={values(1, 2)} asideTrend="negative" variant="danger" />,
        )

        expect(drop.querySelector('svg.text-danger')).not.toBeNull()
        expect(drop.querySelector('.h-20')).not.toBeNull()
        expect(rise.querySelector('svg.text-danger')).not.toBeNull()
        expect(rise.querySelector('.h-8')).not.toBeNull()
    })

    test('marks a drop as good for a negative trend', () => {
        const { container } = render(<MetricCard title="Errors" values={values(2, 1)} asideTrend="negative" />)

        expect(container.querySelector('svg.text-success')).not.toBeNull()
    })

    test.each([
        ['an unchanged latest value', values(1, 1), 'positive' as const],
        ['a single value', values(1), 'positive' as const],
        ['no values', undefined, 'positive' as const],
        ['a disabled trend', values(1, 2), null],
    ])('shows no trend arrow for %s', (_, series, asideTrend) => {
        const { container } = render(<MetricCard title="Rate" value="" values={series} asideTrend={asideTrend} />)

        expect(container.querySelector('svg.size-4')).toBeNull()
    })

    test('prefers an explicit aside over the trend', () => {
        const { container } = render(<MetricCard title="Rate" values={values(1, 2)} aside={<i>custom</i>} separated />)

        expect(container.querySelector('i')?.textContent).toBe('custom')
        expect(container.querySelector('svg.size-4')).toBeNull()
    })
})

describe('AreaChartCard', () => {
    test('renders the title, hint, subtitle, value and an inline legend', () => {
        const { container } = renderWithTooltips(
            <AreaChartCard
                title="Throughput"
                subtitle="ops/s"
                hint="all sinks"
                value="42"
                data={[{ ok: 1, failed: 0 }]}
                series={[
                    { dataKey: 'ok', label: 'Delivered', variant: 'success' },
                    { dataKey: 'failed', variant: 'danger' },
                ]}
            />,
        )

        expect(container.textContent).toContain('Throughput')
        expect(container.textContent).toContain('ops/s')
        expect(container.textContent).toContain('42')
        expect(container.querySelector('.bg-success')?.parentElement?.textContent).toBe('Delivered')
        expect(container.querySelector('.bg-danger')?.parentElement?.textContent).toBe('failed')
    })

    test('omits the legend, value and subtitle when not given', () => {
        const { container } = render(
            <AreaChartCard
                title="Throughput"
                legend={false}
                data={[]}
                series={[{ dataKey: 'ok', variant: 'primary' }]}
            />,
        )

        expect(container.querySelector('.bg-primary')).toBeNull()
        expect(container.querySelector('.text-3xl')).toBeNull()
        expect(container.querySelector('.font-mono.text-xs')).toBeNull()
    })
})

describe('BarChartCard', () => {
    test('lists up to two series inline with hint and subtitle', () => {
        const { container } = renderWithTooltips(
            <BarChartCard
                title="Deliveries"
                subtitle="24h"
                hint="per hour"
                data={[{ ok: 1, failed: 2 }]}
                series={[
                    { dataKey: 'ok', label: 'Delivered', variant: 'success' },
                    { dataKey: 'failed', variant: 'danger' },
                ]}
            />,
        )

        expect(container.textContent).toContain('24h')
        expect(container.textContent).toContain('Delivered')
        expect(container.textContent).not.toContain('series')
    })

    test('collapses more than two series into swatches and a count', () => {
        const series = ['a', 'b', 'c', 'd', 'e', 'f'].map((dataKey) => ({ dataKey, variant: 'amber' as const }))
        const { container } = renderWithTooltips(<BarChartCard title="Deliveries" data={[]} series={series} />)

        expect(container.textContent).toContain('6 series')
        expect(container.querySelectorAll('.bg-amber')).toHaveLength(5)
    })

    test('omits the legend when disabled', () => {
        const { container } = render(
            <BarChartCard
                title="Deliveries"
                legend={false}
                data={[]}
                series={[{ dataKey: 'ok', variant: 'primary' }]}
            />,
        )

        expect(container.querySelector('.bg-primary')).toBeNull()
    })
})
