import type { ComponentProps } from 'react'

import type { Meta, StoryObj } from '@storybook/react'

import { BarChartCard } from './bar-chart-card'

type BarSeries = ComponentProps<typeof BarChartCard>['series'][number]

const VARIANTS: BarSeries['variant'][] = ['primary', 'success', 'amber', 'danger']

const NAMED_SERIES = ['kafka', 'nats', 'rabbit', 'fail']

const DATA_SOURCES = {
    '1 series': 1,
    '2 series': 2,
    '4 series': 4,
    '10 series': 10,
    '40 series': 40,
} as const

type DataSource = keyof typeof DATA_SOURCES

const buildSeries = (count: number): BarSeries[] =>
    Array.from({ length: count }, (_, index) => {
        const name = count <= NAMED_SERIES.length ? NAMED_SERIES[index] : `sink-${index + 1}`
        return { dataKey: name, label: name, variant: VARIANTS[index % VARIANTS.length] }
    })

const buildData = (series: BarSeries[]) =>
    Array.from({ length: 16 }, (_, index) => ({
        time: `16:${String(index * 5).padStart(2, '0')}`,
        ...Object.fromEntries(
            series.map(({ dataKey }, seriesIndex) => [
                dataKey,
                Math.round((120 + ((index * (17 + seriesIndex * 6)) % 60)) / (1 + seriesIndex * 0.5)),
            ]),
        ),
    }))

type StoryArgs = Pick<
    ComponentProps<typeof BarChartCard>,
    'title' | 'subtitle' | 'hint' | 'legend' | 'stacked' | 'tooltip'
> & {
    dataSource: DataSource
}

const meta: Meta<StoryArgs> = {
    title: 'UI Kit/Card/Bar',
    argTypes: {
        title: { name: 'Title', control: 'text', table: { category: 'Content' } },
        subtitle: { name: 'Subtitle', control: 'text', table: { category: 'Content' } },
        hint: { name: 'Hint', control: 'text', table: { category: 'Content' } },
        dataSource: {
            name: 'Data source',
            control: 'select',
            options: Object.keys(DATA_SOURCES),
            table: { category: 'Content' },
        },
        legend: { name: 'Legend', control: 'boolean', table: { category: 'Style' } },
        stacked: { name: 'Stacked', control: 'boolean', table: { category: 'Style' } },
        tooltip: { name: 'Tooltip', control: 'boolean', table: { category: 'Style' } },
    },
    args: {
        title: 'Sink delivery',
        subtitle: 'msg/sec',
        hint: 'Describe the metric shown on this card.',
        dataSource: '4 series',
        legend: true,
        stacked: true,
        tooltip: true,
    },
    render: ({ title, subtitle, hint, dataSource, legend, stacked, tooltip }) => {
        const series = buildSeries(DATA_SOURCES[dataSource])

        return (
            <BarChartCard
                className="w-md"
                title={title}
                subtitle={subtitle || undefined}
                hint={hint || undefined}
                legend={legend}
                stacked={stacked}
                tooltip={tooltip}
                data={buildData(series)}
                series={series}
                xAxisKey="time"
            />
        )
    },
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Bar',
}
