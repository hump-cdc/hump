import type { ComponentProps } from 'react'

import type { Meta, StoryObj } from '@storybook/react'

import { AreaChart } from './area-chart'

type AreaVariant = 'success' | 'primary' | 'danger' | 'amber'
type AreaSeries = ComponentProps<typeof AreaChart>['series']

const toMonthly = (values: number[]): Array<Record<string, number | string>> =>
    values.map((value, index) => ({ month: `M${index + 1}`, value }))

const trendData = toMonthly([46, 62, 55, 78, 70, 90, 84, 108, 96, 120, 112, 134, 150, 128])
const volatileData = toMonthly([80, 40, 95, 50, 110, 35, 90, 60, 120, 45, 100, 70])
const seasonalData = toMonthly([30, 45, 70, 110, 140, 150, 130, 95, 60, 40, 35, 50])

const steadyData = [
    { month: 'Jan', delivered: 80, retried: 24, failed: 8 },
    { month: 'Feb', delivered: 96, retried: 18, failed: 12 },
    { month: 'Mar', delivered: 72, retried: 30, failed: 6 },
    { month: 'Apr', delivered: 110, retried: 22, failed: 14 },
    { month: 'May', delivered: 88, retried: 26, failed: 10 },
    { month: 'Jun', delivered: 124, retried: 16, failed: 9 },
    { month: 'Jul', delivered: 100, retried: 28, failed: 18 },
    { month: 'Aug', delivered: 132, retried: 20, failed: 7 },
]

const spikyData = [
    { month: 'Jan', delivered: 40, retried: 60, failed: 24 },
    { month: 'Feb', delivered: 130, retried: 10, failed: 4 },
    { month: 'Mar', delivered: 56, retried: 44, failed: 30 },
    { month: 'Apr', delivered: 150, retried: 8, failed: 2 },
    { month: 'May', delivered: 48, retried: 52, failed: 28 },
    { month: 'Jun', delivered: 140, retried: 14, failed: 5 },
    { month: 'Jul', delivered: 62, retried: 38, failed: 22 },
    { month: 'Aug', delivered: 134, retried: 12, failed: 6 },
]

const growthData = [
    { month: 'Jan', delivered: 30, retried: 18, failed: 4 },
    { month: 'Feb', delivered: 48, retried: 20, failed: 6 },
    { month: 'Mar', delivered: 66, retried: 22, failed: 7 },
    { month: 'Apr', delivered: 84, retried: 24, failed: 9 },
    { month: 'May', delivered: 102, retried: 26, failed: 11 },
    { month: 'Jun', delivered: 120, retried: 28, failed: 12 },
    { month: 'Jul', delivered: 138, retried: 30, failed: 14 },
    { month: 'Aug', delivered: 156, retried: 32, failed: 16 },
]

const threeSeries = (): AreaSeries => [
    { dataKey: 'delivered', label: 'Delivered', variant: 'success' },
    { dataKey: 'retried', label: 'Retried', variant: 'primary' },
    { dataKey: 'failed', label: 'Failed', variant: 'danger' },
]

const datasets = {
    '1 series — trend': {
        data: trendData,
        series: (variant: AreaVariant): AreaSeries => [{ dataKey: 'value', label: 'Amount', variant }],
    },
    '1 series — volatile': {
        data: volatileData,
        series: (variant: AreaVariant): AreaSeries => [{ dataKey: 'value', label: 'Amount', variant }],
    },
    '1 series — seasonal': {
        data: seasonalData,
        series: (variant: AreaVariant): AreaSeries => [{ dataKey: 'value', label: 'Amount', variant }],
    },
    '3 series — steady': {
        data: steadyData,
        series: threeSeries,
    },
    '3 series — spiky': {
        data: spikyData,
        series: threeSeries,
    },
    '3 series — growth': {
        data: growthData,
        series: threeSeries,
    },
} satisfies Record<
    string,
    { data: Array<Record<string, number | string>>; series: (variant: AreaVariant) => AreaSeries }
>

type StoryArgs = {
    dataset: keyof typeof datasets
    variant: AreaVariant
    stacked: boolean
    tooltip: boolean
    legend: boolean
}

const meta: Meta<StoryArgs> = {
    title: 'UI Kit/Charts/Area Chart',
    argTypes: {
        dataset: {
            name: 'Data source',
            control: 'select',
            options: Object.keys(datasets),
            table: { category: 'Content' },
        },
        variant: {
            name: 'Variant',
            control: 'select',
            options: ['success', 'primary', 'danger', 'amber'],
            table: { category: 'Style' },
        },
        stacked: {
            name: 'Stacked',
            control: 'boolean',
            table: { category: 'Style' },
        },
        tooltip: {
            name: 'Tooltip',
            control: 'boolean',
            table: { category: 'State' },
        },
        legend: {
            name: 'Legend',
            control: 'boolean',
            table: { category: 'State' },
        },
    },
    args: {
        dataset: '1 series — trend',
        variant: 'primary',
        stacked: true,
        tooltip: true,
        legend: false,
    },
    render: ({ dataset, variant, stacked, tooltip, legend }) => {
        const { data, series } = datasets[dataset]

        return (
            <div className="w-full max-w-xl">
                <AreaChart
                    data={data}
                    series={series(variant)}
                    xAxisKey="month"
                    stacked={stacked}
                    tooltip={tooltip}
                    legend={legend}
                />
            </div>
        )
    },
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Area Chart',
}
