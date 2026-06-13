import type { ComponentProps } from 'react'

import type { Meta, StoryObj } from '@storybook/react'
import { RefreshCw } from 'lucide-react'

import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'

import { Header } from './header'
import { HeaderTitle } from './header-title'

type StoryArgs = {
    title: string
    status: string
    statusVariant: ComponentProps<typeof Badge>['variant']
}

const meta: Meta<StoryArgs> = {
    title: 'UI Kit/Page/Header',
    parameters: { layout: 'fullscreen' },
    argTypes: {
        title: { name: 'Title', control: 'text', table: { category: 'Content' } },
        status: { name: 'Status', control: 'text', table: { category: 'Content' } },
        statusVariant: {
            name: 'Status variant',
            control: 'select',
            options: ['neutral', 'healthy', 'lag', 'failed', 'active'],
            table: { category: 'Style' },
        },
    },
    args: {
        title: 'Overview',
        status: 'production',
        statusVariant: 'active',
    },
    render: ({ title, status, statusVariant }) => (
        <Header>
            <div className="flex min-w-0 items-center gap-2.5">
                <HeaderTitle>{title}</HeaderTitle>
                <Badge variant={statusVariant} border dot>
                    {status}
                </Badge>
            </div>
            <Button variant="outline">
                <RefreshCw />
                Refresh
            </Button>
        </Header>
    ),
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Header',
}
