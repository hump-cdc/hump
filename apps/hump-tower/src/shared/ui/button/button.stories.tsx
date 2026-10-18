import type { ComponentProps, MouseEventHandler } from 'react'

import type { Meta, StoryObj } from '@storybook/react'
import { Download, type LucideIcon, Plus, RefreshCw, SlidersHorizontal, Trash2 } from 'lucide-react'
import { fn } from 'storybook/test'

import { Button } from './button'

const icons = {
    none: null,
    Plus,
    Download,
    Trash2,
    SlidersHorizontal,
    RefreshCw,
} satisfies Record<string, LucideIcon | null>

type StoryArgs = Pick<ComponentProps<typeof Button>, 'variant' | 'disabled' | 'nonInteractive' | 'loading'> & {
    label: string
    icon: keyof typeof icons
    onClick: MouseEventHandler<HTMLButtonElement>
}

const meta: Meta<StoryArgs> = {
    title: 'Actions/Button',
    argTypes: {
        label: {
            name: 'Content',
            control: 'text',
            table: { category: 'Content' },
        },
        icon: {
            name: 'Icon',
            control: 'select',
            options: Object.keys(icons),
            table: { category: 'Content' },
        },
        variant: {
            name: 'Variant',
            control: 'select',
            options: ['primary', 'secondary', 'outline', 'ghost', 'danger'],
            table: { category: 'Style' },
        },
        nonInteractive: {
            name: 'Interactive',
            control: 'boolean',
            table: { category: 'Style' },
        },
        disabled: {
            name: 'Disabled',
            control: 'boolean',
            table: { category: 'State' },
        },
        loading: {
            name: 'Loading',
            control: 'boolean',
            table: { category: 'State' },
        },
        onClick: {
            table: { disable: true },
        },
    },
    args: {
        variant: 'primary',
        label: 'Button',
        icon: 'Plus',
        disabled: false,
        nonInteractive: false,
        loading: false,
        onClick: fn(),
    },
    render: ({ variant, label, icon, disabled, nonInteractive, loading, onClick }) => {
        const Icon = icons[icon]

        return (
            <Button
                variant={variant}
                disabled={disabled}
                nonInteractive={nonInteractive}
                loading={loading}
                onClick={onClick}
            >
                {Icon ? <Icon /> : null}
                {label}
            </Button>
        )
    },
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Button',
}
