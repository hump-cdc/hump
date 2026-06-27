import type { ComponentProps } from 'react'

import type { Meta, StoryObj } from '@storybook/react'

import { AreaChartCard } from './area-chart-card'

const COMMIT = [1.4, 1.1, 1.6, 1.3, 1.8, 1.5, 1.9, 1.6, 2.1, 1.7, 1.5, 1.2]
const SENDER = [0.5, 0.4, 0.6, 0.5, 0.7, 0.5, 0.8, 0.6, 0.7, 0.5, 0.4, 0.4]

const DATA = COMMIT.map((commit, index) => ({
    time: `16:${String(index * 5).padStart(2, '0')}`,
    commit,
    sender: SENDER[index],
}))

type StoryArgs = Pick<
    ComponentProps<typeof AreaChartCard>,
    'title' | 'subtitle' | 'hint' | 'legend' | 'stacked' | 'tooltip'
> & {
    value: string
    multiSeries: boolean
}

const meta: Meta<StoryArgs> = {
    title: 'UI Kit/Card/Area',
    argTypes: {
        title: { name: 'Title', control: 'text', table: { category: 'Content' } },
        subtitle: { name: 'Subtitle', control: 'text', table: { category: 'Content' } },
        hint: { name: 'Hint', control: 'text', table: { category: 'Content' } },
        value: { name: 'Value', control: 'text', table: { category: 'Content' } },
        multiSeries: { name: 'Multiple series', control: 'boolean', table: { category: 'Content' } },
        legend: { name: 'Legend', control: 'boolean', table: { category: 'Style' } },
        stacked: { name: 'Stacked', control: 'boolean', table: { category: 'Style' } },
        tooltip: { name: 'Tooltip', control: 'boolean', table: { category: 'Style' } },
    },
    args: {
        title: 'Replication lag',
        subtitle: '',
        hint: 'Describe the metric shown on this card.',
        value: '',
        multiSeries: true,
        legend: true,
        stacked: false,
        tooltip: true,
    },
    render: ({ title, subtitle, hint, value, multiSeries, legend, stacked, tooltip }) => (
        <AreaChartCard
            className="w-md"
            title={title}
            subtitle={subtitle || undefined}
            hint={hint || undefined}
            value={value || undefined}
            legend={legend}
            stacked={stacked}
            tooltip={tooltip}
            data={DATA}
            series={
                multiSeries
                    ? [
                          { dataKey: 'commit', label: 'commit', variant: 'primary' },
                          { dataKey: 'sender', label: 'sender', variant: 'success' },
                      ]
                    : [{ dataKey: 'commit', label: 'commit', variant: 'primary' }]
            }
            xAxisKey="time"
        />
    ),
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Area',
}
