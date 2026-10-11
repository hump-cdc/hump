import type { ComponentProps } from 'react'

import type { Meta, StoryObj } from '@storybook/react'

import { ProgressChart } from './progress-chart'

type StoryArgs = Pick<ComponentProps<typeof ProgressChart>, 'value' | 'showError'> & {
    title: string
    isEmpty: boolean
}

const meta: Meta<StoryArgs> = {
    title: 'Charts/Progress Chart',
    argTypes: {
        title: { name: 'Title', control: 'text', table: { category: 'Content' } },
        value: { name: 'Value', control: { type: 'range', min: 0, max: 100, step: 1 }, table: { category: 'Content' } },
        showError: { name: 'Show error', control: 'boolean', table: { category: 'Style' } },
        isEmpty: { name: 'No value', control: 'boolean', table: { category: 'State' } },
    },
    args: { title: 'Success rate', value: 98, showError: false, isEmpty: false },
    render: ({ title, value, showError, isEmpty }) => (
        <div className="w-75 p-6">
            <ProgressChart title={title} value={isEmpty ? undefined : value} showError={showError} />
        </div>
    ),
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Progress Chart',
}
