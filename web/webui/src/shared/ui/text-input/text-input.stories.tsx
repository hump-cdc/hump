import { type ComponentProps, useState } from 'react'

import type { Meta, StoryObj } from '@storybook/react'
import { Search } from 'lucide-react'
import { fn } from 'storybook/test'

import { TextInput } from './text-input'

type StoryArgs = Pick<ComponentProps<typeof TextInput>, 'placeholder' | 'invalid' | 'disabled' | 'font' | 'type'> & {
    withIcon: boolean
    onValueChange: (value: string) => void
}

const meta: Meta<StoryArgs> = {
    title: 'UI Kit/Inputs/Text Input',
    argTypes: {
        placeholder: { name: 'Placeholder', control: 'text', table: { category: 'Content' } },
        withIcon: { name: 'Leading icon', control: 'boolean', table: { category: 'Content' } },
        font: { name: 'Font', control: 'inline-radio', options: ['mono', 'sans'], table: { category: 'Style' } },
        type: { name: 'Type', control: 'inline-radio', options: ['text', 'password'], table: { category: 'State' } },
        invalid: { name: 'Invalid', control: 'boolean', table: { category: 'State' } },
        disabled: { name: 'Disabled', control: 'boolean', table: { category: 'State' } },
        onValueChange: { table: { disable: true } },
    },
    args: {
        placeholder: 'localhost:9092',
        withIcon: false,
        font: 'mono',
        type: 'text',
        invalid: false,
        disabled: false,
        onValueChange: fn(),
    },
    render: ({ placeholder, withIcon, font, type, invalid, disabled, onValueChange }) => {
        const [value, setValue] = useState('{{ hostname }}')

        return (
            <div className="w-85">
                <TextInput
                    value={value}
                    onValueChange={(next) => {
                        setValue(next)
                        onValueChange(next)
                    }}
                    placeholder={placeholder}
                    type={type}
                    icon={withIcon ? <Search /> : undefined}
                    font={font}
                    invalid={invalid}
                    disabled={disabled}
                />
            </div>
        )
    },
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Text Input',
}
