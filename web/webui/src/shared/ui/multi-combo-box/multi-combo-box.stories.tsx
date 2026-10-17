import { type ComponentProps, useState } from 'react'

import type { Meta, StoryObj } from '@storybook/react'
import { Database } from 'lucide-react'
import { fn } from 'storybook/test'

import { MultiComboBox } from './multi-combo-box'

const sinks = [
    { value: 'postgresql', label: 'PostgreSQL', aside: '1,024' },
    { value: 'clickhouse', label: 'ClickHouse', aside: '842' },
    { value: 'kafka', label: 'Kafka', aside: '318' },
    { value: 's3-archive', label: 'S3 Archive', aside: '56' },
    { value: 'webhook', label: 'Webhook', aside: '24' },
    { value: 'bigquery', label: 'BigQuery', aside: '12' },
    { value: 'snowflake', label: 'Snowflake', aside: '9' },
    { value: 'elasticsearch', label: 'Elasticsearch', aside: '7' },
    { value: 'redis', label: 'Redis', aside: '1,290' },
    { value: 'mysql', label: 'MySQL', aside: '631' },
    { value: 'mongodb', label: 'MongoDB', aside: '408' },
    { value: 'cassandra', label: 'Cassandra', aside: '211' },
    { value: 'rabbitmq', label: 'RabbitMQ', aside: '88' },
    { value: 'nats', label: 'NATS', aside: '41' },
    { value: 'pulsar', label: 'Pulsar', aside: '19' },
    { value: 'dynamodb', label: 'DynamoDB', aside: '6' },
]

type StoryArgs = Pick<ComponentProps<typeof MultiComboBox>, 'placeholder' | 'hint' | 'invalid' | 'disabled'> & {
    withIcon: boolean
    onValueChange: (value: string[]) => void
}

const meta: Meta<StoryArgs> = {
    title: 'Inputs/Multi Combo Box',
    argTypes: {
        placeholder: { name: 'Placeholder', control: 'text', table: { category: 'Content' } },
        withIcon: { name: 'Leading icon', control: 'boolean', table: { category: 'Content' } },
        hint: { name: 'Icon hint', control: 'text', table: { category: 'Content' } },
        invalid: { name: 'Invalid', control: 'boolean', table: { category: 'State' } },
        disabled: { name: 'Disabled', control: 'boolean', table: { category: 'State' } },
        onValueChange: { table: { disable: true } },
    },
    args: {
        placeholder: 'Select sinks…',
        withIcon: false,
        hint: '',
        invalid: false,
        disabled: false,
        onValueChange: fn(),
    },
    render: ({ placeholder, withIcon, hint, invalid, disabled, onValueChange }) => {
        const [value, setValue] = useState(['postgresql', 'clickhouse', 'kafka', 's3-archive', 'webhook'])

        return (
            <div className="flex min-h-96 w-85 items-start">
                <MultiComboBox
                    items={sinks}
                    value={value}
                    onValueChange={(next) => {
                        setValue(next)
                        onValueChange(next)
                    }}
                    placeholder={placeholder}
                    icon={withIcon ? <Database /> : undefined}
                    hint={hint || undefined}
                    invalid={invalid}
                    disabled={disabled}
                />
            </div>
        )
    },
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Multi Combo Box',
}
