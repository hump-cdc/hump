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

type StoryArgs = {
    value: string
}

const meta: Meta<StoryArgs> = {
    title: 'UI Kit/Code',
    argTypes: {
        value: {
            name: 'Source',
            control: 'text',
            table: { category: 'Content' },
        },
    },
    args: {
        value: SAMPLE_YAML,
    },
    render: ({ value }) => (
        <Card>
            <CardContent>
                <Code className="h-full w-full" value={value} />
            </CardContent>
        </Card>
    ),
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Code',
}
