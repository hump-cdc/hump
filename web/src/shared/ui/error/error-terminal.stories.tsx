import type { ComponentProps } from 'react'

import type { Meta, StoryObj } from '@storybook/react'

import { Button } from '../button'
import { ErrorLine } from './error-line'
import { ErrorPrompt } from './error-prompt'
import { ErrorTerminal } from './error-terminal'

type StoryArgs = Pick<ComponentProps<typeof ErrorTerminal>, 'status' | 'title' | 'description' | 'terminalTitle'> & {
    withAction: boolean
}

const meta: Meta<StoryArgs> = {
    title: 'UI Kit/Error Terminal',
    parameters: { disablePadding: true },
    argTypes: {
        title: { name: 'Title', control: 'text', table: { category: 'Content' } },
        description: { name: 'Description', control: 'text', table: { category: 'Content' } },
        status: { name: 'Status', control: 'text', table: { category: 'Content' } },
        terminalTitle: { name: 'Terminal title', control: 'text', table: { category: 'Content' } },
        withAction: { name: 'Action', control: 'boolean', table: { category: 'Style' } },
    },
    args: {
        title: 'We grepped. We found nothing.',
        description: 'Error 404: this page has left the cluster.',
        status: '404 — page not found',
        terminalTitle: 'bash — hump — 80×24',
        withAction: true,
    },
    render: ({ title, description, status, terminalTitle, withAction }) => (
        <div className="h-full w-full">
            <ErrorTerminal
                title={title}
                description={description}
                status={status}
                terminalTitle={terminalTitle}
                action={withAction ? <Button>Go to Dashboard</Button> : undefined}
            >
                <ErrorPrompt>cd /the-page-you-wanted</ErrorPrompt>
                <ErrorLine tone="danger">cd: no such file or directory: /the-page-you-wanted</ErrorLine>

                <ErrorPrompt>grep -r "that page" .</ErrorPrompt>
                <ErrorLine tone="muted">0 results. We looked everywhere. Even prod.</ErrorLine>

                <ErrorPrompt>git blame</ErrorPrompt>
                <ErrorLine tone="warning">fatal: it was DNS. it&apos;s always DNS.</ErrorLine>
            </ErrorTerminal>
        </div>
    ),
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Error Terminal',
}
