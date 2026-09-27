import type { ComponentProps, ReactNode } from 'react'

import type { Meta, StoryObj } from '@storybook/react'

import { MetricCard } from './metric-card'

const SAMPLE_VALUES = {
    up: [16, 18, 15, 20, 19, 22, 21, 24, 23, 26],
    down: [26, 23, 24, 21, 22, 19, 20, 15, 18, 14],
    flat: [12, 9, 14, 11, 16, 13, 10, 15, 12, 12],
} satisfies Record<string, number[]>

const asides = {
    auto: undefined,
    none: null,
    label: <span className="font-medium text-ink-tertiary text-xs">depth</span>,
} satisfies Record<string, ReactNode>

type StoryArgs = Pick<ComponentProps<typeof MetricCard>, 'title' | 'hint' | 'variant' | 'asideTrend' | 'separated'> & {
    value: string
    aside: keyof typeof asides
    direction: keyof typeof SAMPLE_VALUES
    withSparkline: boolean
    footer: string
}

const meta: Meta<StoryArgs> = {
    title: 'UI Kit/Card/Metric',
    argTypes: {
        title: { name: 'Title', control: 'text', table: { category: 'Content' } },
        value: { name: 'Value', control: 'text', table: { category: 'Content' } },
        hint: { name: 'Hint', control: 'text', table: { category: 'Content' } },
        aside: {
            name: 'Aside',
            control: 'select',
            options: Object.keys(asides),
            table: { category: 'Content' },
        },
        footer: { name: 'Footer text', control: 'text', table: { category: 'Content' } },
        variant: {
            name: 'Variant',
            control: 'select',
            options: ['primary', 'success', 'danger', 'amber'],
            table: { category: 'Style' },
        },
        direction: {
            name: 'Trend direction',
            control: 'select',
            options: Object.keys(SAMPLE_VALUES),
            table: { category: 'Style' },
        },
        asideTrend: {
            name: 'Aside trend',
            control: 'inline-radio',
            options: ['positive', 'negative'],
            table: { category: 'Style' },
        },
        withSparkline: { name: 'Sparkline', control: 'boolean', table: { category: 'Style' } },
        separated: { name: 'Header border', control: 'boolean', table: { category: 'Style' } },
    },
    args: {
        title: 'Delivered',
        value: '128,402',
        hint: 'Describe the metric shown on this card.',
        aside: 'auto',
        direction: 'up',
        asideTrend: 'positive',
        footer: 'success total',
        variant: 'success',
        withSparkline: true,
        separated: false,
    },
    render: ({ title, value, hint, aside, direction, asideTrend, footer, variant, withSparkline, separated }) => {
        const values = SAMPLE_VALUES[direction].map((point, index) => ({ date: `t-${index}`, value: point }))

        return (
            <MetricCard
                className="w-72"
                title={title}
                hint={hint || undefined}
                aside={asides[aside]}
                asideTrend={asideTrend}
                value={value}
                values={withSparkline ? values : undefined}
                variant={variant}
                separated={separated}
                footer={footer || undefined}
            />
        )
    },
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Metric',
}
