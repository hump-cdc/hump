import type { ComponentProps } from 'react'

import type { Meta, StoryObj } from '@storybook/react'

import { NumberFlow } from './number-flow'

type StoryArgs = Pick<ComponentProps<typeof NumberFlow>, 'kind' | 'suffix'> & {
    value: number
}

const meta: Meta<StoryArgs> = {
    title: 'UI Kit/Number Flow',
    argTypes: {
        value: { name: 'Value', control: 'number', table: { category: 'Content' } },
        suffix: { name: 'Suffix', control: 'text', table: { category: 'Content' } },
        kind: {
            name: 'Kind',
            control: 'select',
            options: ['integer', 'decimal', 'compact', 'percent', 'bytes', 'latency'],
            table: { category: 'Style' },
        },
    },
    args: { value: 1284300, kind: 'integer', suffix: '' },
    render: ({ value, kind, suffix }) => (
        <span className="font-mono font-semibold text-3xl text-ink tracking-tight">
            <NumberFlow value={value} kind={kind} suffix={suffix || undefined} />
        </span>
    ),
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Number Flow',
}
