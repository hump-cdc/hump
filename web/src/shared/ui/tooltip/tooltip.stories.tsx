import type { ComponentProps } from 'react'

import type { Meta, StoryObj } from '@storybook/react'

import { Tooltip } from './tooltip'
import { TooltipContent } from './tooltip-content'
import { TooltipTrigger } from './tooltip-trigger'

type StoryArgs = Pick<ComponentProps<typeof TooltipContent>, 'variant' | 'side'> & {
    text: string
}

const meta: Meta<StoryArgs> = {
    title: 'UI Kit/Tooltip',
    parameters: { disableSideBySide: true },
    argTypes: {
        text: { name: 'Tooltip', control: 'text', table: { category: 'Content' } },
        variant: {
            name: 'Variant',
            control: 'select',
            options: ['default', 'primary'],
            table: { category: 'Style' },
        },
        side: {
            name: 'Side',
            control: 'select',
            options: ['top', 'right', 'bottom', 'left'],
            table: { category: 'Style' },
        },
    },
    args: {
        text: 'Tooltip label',
        variant: 'default',
        side: 'top',
    },
    render: ({ text, variant, side }) => (
        <div className="flex min-h-40 items-center justify-center">
            <Tooltip>
                <TooltipTrigger className="rounded-md border border-hairline-strong bg-surface-1 px-3 py-1.5 text-base text-ink-muted">
                    Hover me
                </TooltipTrigger>
                <TooltipContent variant={variant} side={side}>
                    {text}
                </TooltipContent>
            </Tooltip>
        </div>
    ),
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Tooltip',
}
