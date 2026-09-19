import type { ComponentProps } from 'react'

import type { Meta, StoryObj } from '@storybook/react'

import { Sheet } from './sheet'
import { SheetContent } from './sheet-content'
import { SheetDescription } from './sheet-description'
import { SheetHeader } from './sheet-header'
import { SheetTitle } from './sheet-title'
import { SheetTrigger } from './sheet-trigger'

type StoryArgs = Pick<ComponentProps<typeof SheetContent>, 'side'> & {
    title: string
    description: string
}

const meta: Meta<StoryArgs> = {
    title: 'UI Kit/Sheet',
    parameters: { disableSideBySide: true },
    argTypes: {
        title: { name: 'Title', control: 'text', table: { category: 'Content' } },
        description: { name: 'Description', control: 'text', table: { category: 'Content' } },
        side: {
            name: 'Side',
            control: 'select',
            options: ['left', 'right', 'top', 'bottom'],
            table: { category: 'Style' },
        },
    },
    args: {
        title: 'Panel title',
        description: 'A slide-in panel anchored to one edge of the viewport.',
        side: 'right',
    },
    render: ({ title, description, side }) => (
        <div className="flex min-h-60 items-center justify-center">
            <Sheet>
                <SheetTrigger className="rounded-lg bg-primary px-3.5 py-2 font-medium text-base text-on-primary">
                    Open sheet
                </SheetTrigger>
                <SheetContent side={side}>
                    <SheetHeader>
                        <SheetTitle>{title}</SheetTitle>
                        <SheetDescription>{description}</SheetDescription>
                    </SheetHeader>
                </SheetContent>
            </Sheet>
        </div>
    ),
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Sheet',
}
