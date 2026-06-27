import type { ComponentProps } from 'react'

import type { Meta, StoryObj } from '@storybook/react'

import { Sink } from './sink'

type StoryArgs = Pick<ComponentProps<typeof Sink>, 'type' | 'host'>

const meta: Meta<StoryArgs> = {
    title: 'UI Kit/Sink',
    argTypes: {
        type: {
            name: 'Type',
            control: 'select',
            options: ['kafka', 'nats-core', 'nats-jetstream', 'http', 'rabbitmq', 'rabbitmq-stream'],
            table: { category: 'Content' },
        },
        host: {
            name: 'Host',
            control: 'text',
            table: { category: 'Content' },
        },
    },
    args: {
        type: 'kafka',
        host: 'localhost:9092',
    },
    render: ({ type, host }) => <Sink type={type} host={host} />,
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Sink',
}
