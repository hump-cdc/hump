import type { Meta, StoryObj } from '@storybook/react'
import { FileX, Inbox, type LucideIcon, SearchX } from 'lucide-react'

import { NoPayloadPlaceholder } from './no-payload-placeholder'
import { Placeholder } from './placeholder'

const icons = {
    none: null,
    FileX,
    Inbox,
    SearchX,
} satisfies Record<string, LucideIcon | null>

type StoryArgs = {
    title: string
    description: string
    icon: keyof typeof icons
}

const meta: Meta<StoryArgs> = {
    title: 'UI Kit/Placeholder',
    argTypes: {
        title: { name: 'Title', control: 'text', table: { category: 'Content' } },
        description: { name: 'Description', control: 'text', table: { category: 'Content' } },
        icon: { name: 'Icon', control: 'select', options: Object.keys(icons), table: { category: 'Content' } },
    },
    args: { title: 'No payload', description: 'No message body for the operation.', icon: 'FileX' },
    render: ({ title, description, icon }) => {
        const Icon = icons[icon]

        return (
            <div className="flex h-60 w-90">
                <Placeholder icon={Icon ? <Icon /> : undefined} title={title} description={description} />
            </div>
        )
    },
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Placeholder',
}

export const NoPayload: StoryObj<StoryArgs> = {
    name: 'No Payload',
    parameters: { controls: { disable: true } },
    render: () => (
        <div className="flex h-60 w-90">
            <NoPayloadPlaceholder />
        </div>
    ),
}
