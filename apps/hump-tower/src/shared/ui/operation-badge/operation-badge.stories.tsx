import type { ComponentProps } from 'react'

import type { Meta, StoryObj } from '@storybook/react'

import { OperationBadge } from './operation-badge'

type StoryArgs = Pick<ComponentProps<typeof OperationBadge>, 'op'>

const meta: Meta<StoryArgs> = {
    title: 'Content/Operation Badge',
    argTypes: {
        op: {
            name: 'Operation',
            control: 'select',
            options: ['insert', 'update', 'delete', 'truncate'],
            table: { category: 'Content' },
        },
    },
    args: {
        op: 'insert',
    },
    render: ({ op }) => <OperationBadge op={op} />,
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Operation Badge',
}
