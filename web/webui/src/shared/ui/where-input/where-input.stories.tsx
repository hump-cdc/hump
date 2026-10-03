import { type ComponentProps, useState } from 'react'

import type { Meta, StoryObj } from '@storybook/react'
import { fn } from 'storybook/test'

import { WhereInput } from './where-input'

type StoryArgs = Pick<
    ComponentProps<typeof WhereInput>,
    'placeholder' | 'suggestion' | 'fields' | 'invalid' | 'disabled'
> & {
    onValueChange: (value: string) => void
}

const meta: Meta<StoryArgs> = {
    title: 'UI Kit/Inputs/Where Input',
    argTypes: {
        placeholder: { name: 'Placeholder', control: 'text', table: { category: 'Content' } },
        suggestion: { name: 'Suggestion', control: 'text', table: { category: 'Content' } },
        fields: { name: 'Fields', control: 'object', table: { category: 'Content' } },
        invalid: { name: 'Invalid', control: 'boolean', table: { category: 'State' } },
        disabled: { name: 'Disabled', control: 'boolean', table: { category: 'State' } },
        onValueChange: { table: { disable: true } },
    },
    args: {
        placeholder: '(no filter)',
        suggestion: "status = 'active'",
        fields: [
            { name: 'id', type: 'bigint' },
            { name: 'status', type: 'text' },
            { name: 'email', type: 'character varying(255)' },
            { name: 'amount', type: 'numeric(12,2)' },
            { name: 'metadata', type: 'jsonb' },
            { name: 'created_at', type: 'timestamp with time zone' },
            { name: 'UpdatedAt', type: 'timestamp without time zone' },
        ],
        invalid: false,
        disabled: false,
        onValueChange: fn(),
    },
    render: ({ placeholder, suggestion, fields, invalid, disabled, onValueChange }) => {
        const [value, setValue] = useState('')

        return (
            <div className="w-85">
                <WhereInput
                    value={value}
                    onValueChange={(next) => {
                        setValue(next)
                        onValueChange(next)
                    }}
                    placeholder={placeholder}
                    suggestion={suggestion}
                    fields={fields}
                    invalid={invalid}
                    disabled={disabled}
                />
            </div>
        )
    },
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Where Input',
}
