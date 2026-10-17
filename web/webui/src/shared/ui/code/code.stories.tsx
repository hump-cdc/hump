import type { ComponentProps } from 'react'

import type { Meta, StoryObj } from '@storybook/react'

import { Card, CardContent } from '../card'
import { Code } from './code'

const SAMPLE_YAML = `sinks:
    - name: kafka1
      kafka:
          url: "localhost:9092"
          topic: "{{.Table}}"
    - name: nats-core
      nats-core:
          url: "nats://localhost:4222"
          subject: "hump.{{.Schema}}.{{.Table}}"`

type StoryArgs = Pick<ComponentProps<typeof Code>, 'lineNumbers' | 'format'> & {
    value: string
}

const meta: Meta<StoryArgs> = {
    title: 'Content/Code',
    argTypes: {
        value: {
            name: 'Source',
            control: 'text',
            table: { category: 'Content' },
        },
        lineNumbers: { name: 'Line numbers', control: 'boolean', table: { category: 'Style' } },
        format: { name: 'Format', control: 'boolean', table: { category: 'Style' } },
    },
    args: {
        value: SAMPLE_YAML,
        lineNumbers: true,
        format: false,
    },
    render: ({ value, lineNumbers, format }) => (
        <Card>
            <CardContent>
                <Code className="h-full w-full" value={value} lineNumbers={lineNumbers} format={format} />
            </CardContent>
        </Card>
    ),
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Code',
}
