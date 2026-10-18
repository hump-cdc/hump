import { type ComponentProps, useState } from 'react'

import type { Meta, StoryObj } from '@storybook/react'
import { fn } from 'storybook/test'

import { Checkbox } from './checkbox'

type CheckedState = ComponentProps<typeof Checkbox>['checked']

type StoryArgs = Pick<ComponentProps<typeof Checkbox>, 'disabled' | 'invalid'> & {
    indeterminate: boolean
    onCheckedChange: (checked: CheckedState) => void
}

const next = (current: CheckedState, withIndeterminate: boolean): CheckedState => {
    if (withIndeterminate) {
        if (current === false) return true
        if (current === true) return 'indeterminate'
        return false
    }
    return current !== true
}

const meta: Meta<StoryArgs> = {
    title: 'Inputs/Checkbox',
    argTypes: {
        indeterminate: { name: 'Indeterminate state', control: 'boolean', table: { category: 'State' } },
        invalid: { name: 'Invalid', control: 'boolean', table: { category: 'State' } },
        disabled: { name: 'Disabled', control: 'boolean', table: { category: 'State' } },
        onCheckedChange: { table: { disable: true } },
    },
    args: {
        indeterminate: true,
        invalid: false,
        disabled: false,
        onCheckedChange: fn(),
    },
    render: ({ indeterminate, invalid, disabled, onCheckedChange }) => {
        const [checked, setChecked] = useState<CheckedState>('indeterminate')

        return (
            <Checkbox
                checked={checked}
                onCheckedChange={() => {
                    const value = next(checked, indeterminate)
                    setChecked(value)
                    onCheckedChange(value)
                }}
                invalid={invalid}
                disabled={disabled}
            />
        )
    },
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Checkbox',
}
