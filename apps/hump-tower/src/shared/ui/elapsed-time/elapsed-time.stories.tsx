import type { Meta, StoryObj } from '@storybook/react'

import { ElapsedTime } from './elapsed-time'

type StoryArgs = {
    seconds: number
    running: boolean
}

const meta: Meta<StoryArgs> = {
    title: 'Content/Elapsed Time',
    argTypes: {
        seconds: { name: 'Elapsed seconds', control: 'number', table: { category: 'Content' } },
        running: { name: 'Running', control: 'boolean', table: { category: 'State' } },
    },
    args: { seconds: 3725, running: true },
    render: ({ seconds, running }) => {
        const startedAt = new Date(Date.now() - seconds * 1000)
        const finishedAt = running ? undefined : new Date(startedAt.getTime() + seconds * 1000)

        return (
            <span className="font-semibold text-3xl text-ink tracking-tight">
                <ElapsedTime startedAt={startedAt} finishedAt={finishedAt} />
            </span>
        )
    },
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Elapsed Time',
}
