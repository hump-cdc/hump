import type { ComponentProps } from 'react'

import type { Meta, StoryObj } from '@storybook/react'

import { ProgressBar } from './progress-bar'

type StoryArgs = Pick<ComponentProps<typeof ProgressBar>, 'value' | 'variant' | 'size'>

const meta: Meta<StoryArgs> = {
    title: 'Placeholders/Progress Bar',
    argTypes: {
        value: { name: 'Value', control: { type: 'range', min: 0, max: 100, step: 1 }, table: { category: 'Content' } },
        variant: {
            name: 'Variant',
            control: 'inline-radio',
            options: ['primary', 'success', 'neutral'],
            table: { category: 'Style' },
        },
        size: { name: 'Size', control: 'inline-radio', options: ['sm', 'md'], table: { category: 'Style' } },
    },
    args: {
        value: 58,
        variant: 'primary',
        size: 'md',
    },
    render: ({ value, variant, size }) => (
        <div className="w-96">
            <ProgressBar value={value} variant={variant} size={size} />
        </div>
    ),
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Progress Bar',
}
