import type { ComponentProps } from 'react'

import type { Meta, StoryObj } from '@storybook/react'
import { fn } from 'storybook/test'

import { Tabs } from './tabs'
import { TabsList } from './tabs-list'
import { TabsTrigger } from './tabs-trigger'

type StoryArgs = Pick<ComponentProps<typeof Tabs>, 'defaultValue'> &
    Pick<ComponentProps<typeof TabsList>, 'variant'> & {
        onValueChange: (value: string) => void
    }

const meta: Meta<StoryArgs> = {
    title: 'UI Kit/Tabs',
    argTypes: {
        defaultValue: {
            name: 'Default tab',
            control: 'select',
            options: ['1m', '5m', '10m'],
            table: { category: 'State' },
        },
        variant: {
            name: 'Variant',
            control: 'select',
            options: ['pill', 'underline'],
            table: { category: 'Style' },
        },
        onValueChange: { table: { disable: true } },
    },
    args: {
        defaultValue: '1m',
        variant: 'pill',
        onValueChange: fn(),
    },
    render: ({ defaultValue, variant, onValueChange }) => (
        <Tabs defaultValue={defaultValue} onValueChange={onValueChange} className="w-80">
            <TabsList variant={variant}>
                <TabsTrigger value="1m">1m</TabsTrigger>
                <TabsTrigger value="5m">5m</TabsTrigger>
                <TabsTrigger value="10m">10m</TabsTrigger>
                <TabsTrigger value="1h" disabled>
                    1h
                </TabsTrigger>
            </TabsList>
        </Tabs>
    ),
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Tabs',
}
