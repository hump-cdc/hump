import type { Meta, StoryObj } from '@storybook/react'

import { Mark } from './mark'

type StoryArgs = {
    before: string
    label: string
    after: string
}

const meta: Meta<StoryArgs> = {
    title: 'Content/Mark',
    argTypes: {
        before: { name: 'Text before', control: 'text', table: { category: 'Content' } },
        label: { name: 'Content', control: 'text', table: { category: 'Content' } },
        after: { name: 'Text after', control: 'text', table: { category: 'Content' } },
    },
    args: {
        before: 'Add an optional ',
        label: 'WHERE',
        after: ' condition per table to snapshot only matching rows.',
    },
    render: ({ before, label, after }) => (
        <p className="text-ink-muted text-sm">
            {before}
            <Mark>{label}</Mark>
            {after}
        </p>
    ),
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Mark',
}
