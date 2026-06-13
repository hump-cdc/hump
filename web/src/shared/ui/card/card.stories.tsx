import type { ReactNode } from 'react'

import type { Meta, StoryObj } from '@storybook/react'

import { cn } from '@/shared/utils'

import { Badge } from '../badge'
import { Button } from '../button'
import { Card } from './card'
import { CardContent } from './card-content'
import { CardFooter } from './card-footer'
import { CardHeader } from './card-header'

const headerAsides = {
    none: null,
    link: <span className="text-sm font-medium text-ink-subtle">View all</span>,
    badge: (
        <Badge variant="healthy" dot>
            Healthy
        </Badge>
    ),
} satisfies Record<string, ReactNode>

type StoryArgs = {
    title: string
    headerAside: keyof typeof headerAsides
    separated: boolean
    withFooter: boolean
}

const rows = [
    { label: 'First metric', value: '6' },
    { label: 'Second metric', value: '12 ms' },
    { label: 'Third metric', value: '48.2k rows/s' },
]

const meta: Meta<StoryArgs> = {
    title: 'UI Kit/Card',
    argTypes: {
        title: {
            name: 'Title',
            control: 'text',
            table: { category: 'Content' },
        },
        headerAside: {
            name: 'Header aside',
            control: 'select',
            options: Object.keys(headerAsides),
            table: { category: 'Content' },
        },
        separated: {
            name: 'Separated',
            control: 'boolean',
            table: { category: 'Style' },
        },
        withFooter: {
            name: 'Footer',
            control: 'boolean',
            table: { category: 'Style' },
        },
    },
    args: {
        title: 'Card title',
        headerAside: 'link',
        separated: true,
        withFooter: false,
    },
    render: ({ title, headerAside, separated, withFooter }) => (
        <Card separated={separated} className="w-90">
            <CardHeader aside={headerAsides[headerAside]}>
                <span className="text-md font-semibold text-ink">{title}</span>
            </CardHeader>

            <CardContent>
                {separated ? (
                    <div className="flex flex-col">
                        {rows.map((row, index) => (
                            <div
                                key={row.label}
                                className={cn(
                                    'flex items-center justify-between px-4 py-2.75',
                                    index < rows.length - 1 && 'border-b border-hairline',
                                )}
                            >
                                <span className="text-base text-ink-subtle">{row.label}</span>
                                <span className="font-mono text-base text-ink">{row.value}</span>
                            </div>
                        ))}
                    </div>
                ) : (
                    <div className="flex flex-col gap-2">
                        <p className="text-base text-ink-muted">Content slot — drop any generic content here.</p>
                        <p className="text-sm text-ink-subtle">
                            With a non-separated card the body shares the card's even inset on every side.
                        </p>
                    </div>
                )}
            </CardContent>

            {withFooter ? (
                <CardFooter aside={<Button variant="secondary">Action</Button>}>
                    <span className="text-sm text-ink-subtle">Footer slot</span>
                </CardFooter>
            ) : null}
        </Card>
    ),
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Card',
}
