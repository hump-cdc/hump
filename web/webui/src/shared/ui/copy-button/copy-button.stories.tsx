import type { ComponentProps } from 'react'

import type { Meta, StoryObj } from '@storybook/react'

import { CopyButton } from './copy-button'

type StoryArgs = Pick<ComponentProps<typeof CopyButton>, 'value' | 'label'>

const meta: Meta<StoryArgs> = {
    title: 'UI Kit/Copy Button',
    argTypes: {
        value: { name: 'Value', control: 'text', table: { category: 'Content' } },
        label: { name: 'Label', control: 'text', table: { category: 'Content' } },
    },
    args: { value: 'orders.public.v1', label: 'Copy value' },
    render: ({ value, label }) => <CopyButton value={value} label={label} />,
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'CopyButton',
}
