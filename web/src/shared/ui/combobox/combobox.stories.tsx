import { useState } from 'react'

import type { Meta, StoryObj } from '@storybook/react'
import { Check, ChevronsUpDown, Database } from 'lucide-react'
import { fn } from 'storybook/test'

import {
    Combobox,
    ComboboxChips,
    ComboboxChipsInput,
    ComboboxClear,
    ComboboxContent,
    ComboboxEmpty,
    ComboboxItem,
    ComboboxList,
    useComboboxAnchor,
} from '@/shared/ui/combobox'

type Sink = { value: string; label: string; aside: string }

const sinks: Sink[] = [
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
]

type StoryArgs = {
    placeholder: string
    emptyText: string
    withIcon: boolean
    invalid: boolean
    disabled: boolean
    onValueChange: (value: Sink | null) => void
}

const meta: Meta<StoryArgs> = {
    title: 'UI Kit/Inputs/Combobox',
    argTypes: {
        placeholder: { name: 'Placeholder', control: 'text', table: { category: 'Content' } },
        emptyText: { name: 'Empty text', control: 'text', table: { category: 'Content' } },
        withIcon: { name: 'Leading icon', control: 'boolean', table: { category: 'Content' } },
        invalid: { name: 'Invalid', control: 'boolean', table: { category: 'State' } },
        disabled: { name: 'Disabled', control: 'boolean', table: { category: 'State' } },
        onValueChange: { table: { disable: true } },
    },
    args: {
        placeholder: 'Select a sink…',
        emptyText: 'No sinks found.',
        withIcon: false,
        invalid: false,
        disabled: false,
        onValueChange: fn(),
    },
    render: ({ placeholder, emptyText, withIcon, invalid, disabled, onValueChange }) => {
        const [value, setValue] = useState<Sink | null>(sinks[0])
        const anchor = useComboboxAnchor()

        return (
            <div className="flex min-h-96 w-85 items-start">
                <Combobox
                    items={sinks}
                    value={value}
                    onValueChange={(next: Sink | null) => {
                        setValue(next)
                        onValueChange(next)
                    }}
                    itemToStringLabel={(item: Sink) => item.label}
                    isItemEqualToValue={(a: Sink, b: Sink) => a.value === b.value}
                    disabled={disabled}
                >
                    <ComboboxChips ref={anchor} aria-invalid={invalid || undefined}>
                        {withIcon && (
                            <span className="flex shrink-0 items-center text-ink-subtle [&_svg]:size-4">
                                <Database />
                            </span>
                        )}
                        <ComboboxChipsInput placeholder={placeholder} />
                        {value && <ComboboxClear disabled={disabled} />}
                        <ChevronsUpDown className="pointer-events-none size-4 shrink-0 text-ink-subtle" />
                    </ComboboxChips>

                    <ComboboxContent anchor={anchor}>
                        <ComboboxList>
                            {(item: Sink) => (
                                <ComboboxItem key={item.value} value={item}>
                                    <span className="flex-1 truncate text-md font-medium text-ink">{item.label}</span>
                                    <span className="shrink-0 font-mono text-sm text-ink-tertiary tabular-nums">
                                        {item.aside}
                                    </span>
                                    <Check className="size-4 shrink-0 text-primary opacity-0 group-data-[selected]/combobox-item:opacity-100" />
                                </ComboboxItem>
                            )}
                        </ComboboxList>
                        <ComboboxEmpty>{emptyText}</ComboboxEmpty>
                    </ComboboxContent>
                </Combobox>
            </div>
        )
    },
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Combobox',
}
