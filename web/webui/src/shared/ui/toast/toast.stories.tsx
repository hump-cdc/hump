import type { ComponentProps } from 'react'

import type { Meta, StoryObj } from '@storybook/react'
import { fn } from 'storybook/test'

import { Button } from '../button'
import { Toaster, toast } from './toast'

const variants = {
    default: toast,
    success: toast.success,
    danger: toast.error,
    amber: toast.warning,
    loading: toast.loading,
}

type StoryArgs = Pick<ComponentProps<typeof Toaster>, 'position'> & {
    title: string
    description: string
    variant: keyof typeof variants
    actionLabel: string
    onAction: () => void
}

const meta: Meta<StoryArgs> = {
    title: 'UI Kit/Toast',
    parameters: { disableSideBySide: true },
    argTypes: {
        title: { name: 'Title', control: 'text', table: { category: 'Content' } },
        description: { name: 'Description', control: 'text', table: { category: 'Content' } },
        actionLabel: { name: 'Action', control: 'text', table: { category: 'Content' } },
        variant: {
            name: 'Variant',
            control: 'select',
            options: Object.keys(variants),
            table: { category: 'Style' },
        },
        position: {
            name: 'Position',
            control: 'select',
            options: ['top-left', 'top-center', 'top-right', 'bottom-left', 'bottom-center', 'bottom-right'],
            table: { category: 'Style' },
        },
        onAction: { table: { disable: true } },
    },
    args: {
        title: 'Sink reconnected',
        description: 'Delivery resumed from the last confirmed position.',
        actionLabel: '',
        variant: 'default',
        position: 'bottom-center',
        onAction: fn(),
    },
    render: ({ title, description, variant, actionLabel, onAction, position }) => (
        <div className="flex min-h-40 items-center justify-center">
            <Button
                variant="outline"
                onClick={() =>
                    variants[variant](title, {
                        description: description || undefined,
                        action: actionLabel ? { label: actionLabel, onClick: onAction } : undefined,
                    })
                }
            >
                Show toast
            </Button>
            <Toaster position={position} />
        </div>
    ),
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Toast',
}
