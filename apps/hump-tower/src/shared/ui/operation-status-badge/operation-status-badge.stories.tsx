import type { ComponentProps } from 'react'

import type { Meta, StoryObj } from '@storybook/react'

import { OperationStatusBadge } from './operation-status-badge'

type StoryArgs = Pick<ComponentProps<typeof OperationStatusBadge>, 'status' | 'attempts' | 'variant' | 'withAttempts'>

const meta: Meta<StoryArgs> = {
    title: 'Content/Operation Status Badge',
    argTypes: {
        status: {
            name: 'Status',
            control: 'select',
            options: ['delivered', 'failed'],
            table: { category: 'Content' },
        },
        attempts: { name: 'Attempts', control: { type: 'number', min: 1 }, table: { category: 'Content' } },
        variant: {
            name: 'Variant',
            control: 'select',
            options: ['full', 'compact'],
            table: { category: 'Style' },
        },
        withAttempts: { name: 'With attempts', control: 'boolean', table: { category: 'Style' } },
    },
    args: {
        status: 'delivered',
        attempts: 1,
        variant: 'full',
        withAttempts: false,
    },
    render: ({ status, attempts, variant, withAttempts }) => (
        <OperationStatusBadge status={status} attempts={attempts} variant={variant} withAttempts={withAttempts} />
    ),
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Operation Status Badge',
}
