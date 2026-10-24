import type { ChartPoint } from '@tanstack/charts'

import type { FC } from 'react'

import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { cleanup, render } from '@testing-library/react'

import { useChart } from './chart-context'
import {
    type ChartConfig,
    ChartContainer,
    ChartLegend,
    type ChartRow,
    ChartTooltipContent,
    chartTooltipOptions,
} from './index'

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(cleanup)

/** Icon a config entry can carry in place of the color swatch. */
const Icon: FC = () => <svg data-testid="icon" />

const config: ChartConfig = {
    ok: { label: 'Delivered', color: 'var(--success)' },
    lag: { label: 'Lagging', icon: Icon, theme: { light: 'red', dark: 'blue' } },
}

/** Builds a focused tooltip point for one series at one x step. */
const createPoint = (
    datum: Partial<ChartRow> & Pick<ChartRow, 'series' | 'value'>,
    y = 0,
): ChartPoint<ChartRow, number, number> =>
    ({
        x: 1,
        y,
        datum: { key: datum.series, index: 1, label: 'Mon', start: 0, end: datum.value, ...datum },
    }) as unknown as ChartPoint<ChartRow, number, number>

describe('ChartContainer', () => {
    test('scopes the color variables to the given id and renders the legend', () => {
        const { container } = render(
            <ChartContainer id="x" config={config} legend>
                {(chartId) => <span data-testid="plot">{chartId}</span>}
            </ChartContainer>,
        )
        const css = container.querySelector('style')?.innerHTML ?? ''

        expect(container.querySelector('[data-chart=chart-x]')).not.toBeNull()
        expect(container.textContent).toContain('chart-x')
        expect(css).toContain('--color-ok: var(--success);')
        expect(css).toContain('.dark [data-chart=chart-x]')
        expect(css).toContain('--color-lag: blue;')
        expect(container.querySelector('[data-testid=icon]')).not.toBeNull()
    })

    test('renders no style and no legend without colored series', () => {
        const { container } = render(
            <ChartContainer config={{ plain: { label: 'Plain' } }}>{() => null}</ChartContainer>,
        )

        expect(container.querySelector('style')).toBeNull()
        expect(container.textContent).toBe('')
    })
})

describe('ChartLegend', () => {
    test('draws swatches instead of icons with hideIcon', () => {
        const { container } = render(<ChartContainer config={config}>{() => <ChartLegend hideIcon />}</ChartContainer>)

        expect(container.querySelector('[data-testid=icon]')).toBeNull()
        expect(container.querySelector('.justify-center')?.textContent).toBe('DeliveredLagging')
    })

    test('renders nothing for an empty config', () => {
        const { container } = render(<ChartContainer config={{}}>{() => <ChartLegend />}</ChartContainer>)

        expect(container.querySelector('[data-slot=chart] > div')?.innerHTML).toBe('')
    })
})

describe('useChart', () => {
    test('throws outside a ChartContainer', () => {
        const Probe: FC = () => {
            useChart()
            return null
        }

        expect(() => render(<Probe />)).toThrow('useChart must be used within a <ChartContainer />')
    })
})

describe('chartTooltipOptions.anchor', () => {
    test('floats above the highest point of the first x', () => {
        const points = [createPoint({ series: 'ok', value: 1 }, 30), createPoint({ series: 'lag', value: 2 }, 10)]

        expect(chartTooltipOptions.anchor(points)).toEqual({ x: 1, y: 10 })
    })

    test('returns undefined without points', () => {
        expect(chartTooltipOptions.anchor([])).toBeUndefined()
    })
})

describe('ChartTooltipContent', () => {
    test('orders rows by series and falls back to the key without a config entry', () => {
        const points = [
            createPoint({ series: 'unknown', value: 1.5 }),
            createPoint({ series: 'lag', value: 2 }),
            createPoint({ series: 'ok', value: 3 }),
        ]
        const { container } = render(
            <ChartContainer config={config}>
                {() => <ChartTooltipContent points={points} seriesOrder={['ok', 'lag', 'unknown']} />}
            </ChartContainer>,
        )
        const labels = [...container.querySelectorAll('span.truncate')].map((node) => node.textContent)

        expect(labels).toEqual(['Delivered', 'Lagging', 'unknown'])
        expect(container.textContent).toContain('Mon')
        expect(container.querySelector('[data-testid=icon]')).not.toBeNull()
    })

    test('omits the label row when the step has none and applies the value props', () => {
        const { container } = render(
            <ChartContainer config={config}>
                {() => (
                    <ChartTooltipContent
                        points={[createPoint({ series: 'ok', value: 4, label: undefined })]}
                        seriesOrder={['ok']}
                        valueProps={{ kind: 'decimal', suffix: 'ms' }}
                    />
                )}
            </ChartContainer>,
        )

        expect(container.querySelector('.font-medium:not(span)')).toBeNull()
        expect(container.textContent).toContain('Delivered')
    })

    test('renders nothing without points', () => {
        const { container } = render(
            <ChartContainer config={config}>
                {() => <ChartTooltipContent points={[]} seriesOrder={[]} />}
            </ChartContainer>,
        )

        expect(container.querySelector('.grid')).toBeNull()
    })
})
