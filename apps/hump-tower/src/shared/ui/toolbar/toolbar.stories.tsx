import type { ComponentProps } from 'react'

import type { Meta, StoryObj } from '@storybook/react'
import { Search } from 'lucide-react'

import { MultiComboBox } from '@/shared/ui/multi-combo-box'
import { TextInput } from '@/shared/ui/text-input'

import { Toolbar } from './toolbar'

const statusItems = [
    { value: 'delivered', label: 'delivered' },
    { value: 'failed', label: 'failed' },
    { value: 'retrying', label: 'retrying' },
]

type StoryArgs = Pick<ComponentProps<typeof Toolbar>, 'label'>

const meta: Meta<StoryArgs> = {
    title: 'Layout/Toolbar',
    argTypes: {
        label: { name: 'Mobile label', control: 'text', table: { category: 'Content' } },
    },
    args: {
        label: 'Filters',
    },
    render: ({ label }) => (
        <Toolbar label={label}>
            <MultiComboBox items={statusItems} placeholder="Status" className="w-72" />
            <TextInput placeholder="Search…" icon={<Search />} font="sans" className="w-full" />
        </Toolbar>
    ),
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Toolbar',
}
