import type { ComponentProps } from 'react'

import type { Meta, StoryObj } from '@storybook/react'

import { InfoCard } from './info-card'

type StoryArgs = Pick<ComponentProps<typeof InfoCard>, 'title' | 'variant'> & {
    content: string
    footer: string
}

const meta: Meta<StoryArgs> = {
    title: 'UI Kit/Card/Info',
    argTypes: {
        title: { name: 'Title', control: 'text', table: { category: 'Content' } },
        content: { name: 'Content', control: 'text', table: { category: 'Content' } },
        footer: { name: 'Footer', control: 'text', table: { category: 'Content' } },
        variant: {
            name: 'Variant',
            control: 'inline-radio',
            options: ['neutral', 'success', 'failed', 'active', 'lag'],
            table: { category: 'Style' },
        },
    },
    args: { variant: 'neutral', title: 'Sink', content: 'kafka', footer: 'kafka · localhost:9092' },
    render: ({ variant, title, content, footer }) => (
        <div className="w-75 p-6">
            <InfoCard variant={variant} title={title} footer={footer || undefined}>
                {content}
            </InfoCard>
        </div>
    ),
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Info',
}
