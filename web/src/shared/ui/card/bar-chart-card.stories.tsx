import type { ComponentProps } from 'react'

import type { Meta, StoryObj } from '@storybook/react'

import { BarChartCard } from './bar-chart-card'

const DATA = Array.from({ length: 16 }, (_, index) => ({
    time: `16:${String(index * 5).padStart(2, '0')}`,
    kafka: 120 + ((index * 17) % 60),
    nats: 80 + ((index * 11) % 40),
    rabbit: 60 + ((index * 7) % 30),
    fail: index % 5 === 0 ? 12 : 4,
}))

type StoryArgs = Pick<
    ComponentProps<typeof BarChartCard>,
    'title' | 'subtitle' | 'hint' | 'legend' | 'stacked' | 'tooltip'
> & {
    multiSeries: boolean
}

const meta: Meta<StoryArgs> = {
    title: 'UI Kit/Card/Bar',
    argTypes: {
        title: { name: 'Title', control: 'text', table: { category: 'Content' } },
        subtitle: { name: 'Subtitle', control: 'text', table: { category: 'Content' } },
        hint: { name: 'Hint', control: 'text', table: { category: 'Content' } },
        multiSeries: { name: 'Multiple series', control: 'boolean', table: { category: 'Content' } },
        legend: { name: 'Legend', control: 'boolean', table: { category: 'Style' } },
        stacked: { name: 'Stacked', control: 'boolean', table: { category: 'Style' } },
        tooltip: { name: 'Tooltip', control: 'boolean', table: { category: 'Style' } },
    },
    args: {
        title: 'Sink delivery',
        subtitle: 'msg/sec',
        hint: 'Describe the metric shown on this card.',
        multiSeries: true,
        legend: true,
        stacked: true,
        tooltip: true,
    },
    render: ({ title, subtitle, hint, multiSeries, legend, stacked, tooltip }) => (
        <BarChartCard
            className="w-[28rem]"
            title={title}
            subtitle={subtitle || undefined}
            hint={hint || undefined}
            legend={legend}
            stacked={stacked}
            tooltip={tooltip}
            data={DATA}
            series={
                multiSeries
                    ? [
                          { dataKey: 'kafka', label: 'kafka', variant: 'primary' },
                          { dataKey: 'nats', label: 'nats', variant: 'success' },
                          { dataKey: 'rabbit', label: 'rabbit', variant: 'amber' },
                          { dataKey: 'fail', label: 'fail', variant: 'danger' },
                      ]
                    : [{ dataKey: 'kafka', label: 'kafka', variant: 'primary' }]
            }
            xAxisKey="time"
        />
    ),
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Bar',
}
