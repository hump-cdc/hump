import type { ComponentProps } from 'react'

import type { Meta, StoryObj } from '@storybook/react'
import { Database, GitBranch, type LucideIcon, Server, Tag } from 'lucide-react'

import { Badge } from './badge'

const icons = {
    none: null,
    Database,
    Server,
    GitBranch,
    Tag,
} satisfies Record<string, LucideIcon | null>

type StoryArgs = Pick<ComponentProps<typeof Badge>, 'variant' | 'dot' | 'dotAnimated' | 'border'> & {
    label: string
    title: string
    icon: keyof typeof icons
}

const meta: Meta<StoryArgs> = {
    title: 'UI Kit/Badge',
    argTypes: {
        label: {
            name: 'Content',
            control: 'text',
            table: { category: 'Content' },
        },
        title: {
            name: 'Title',
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
            options: ['neutral', 'healthy', 'lag', 'failed', 'active'],
            table: { category: 'Style' },
        },
        dot: {
            name: 'Dot',
            control: 'boolean',
            table: { category: 'Style' },
        },
        dotAnimated: {
            name: 'Dot animated',
            control: 'boolean',
            table: { category: 'Style' },
        },
        border: {
            name: 'Border',
            control: 'boolean',
            table: { category: 'Style' },
        },
    },
    args: {
        variant: 'neutral',
        label: 'app',
        title: 'db',
        icon: 'Database',
        dot: false,
        dotAnimated: false,
        border: false,
    },
    render: ({ variant, label, title, icon, dot, dotAnimated, border }) => {
        const Icon = icons[icon]

        return (
            <Badge
                variant={variant}
                dot={dot}
                dotAnimated={dotAnimated}
                border={border}
                icon={Icon ? <Icon /> : undefined}
                title={title}
            >
                {label}
            </Badge>
        )
    },
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Badge',
}
