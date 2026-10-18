import { useState } from 'react'

import type { Meta, StoryObj } from '@storybook/react'
import { CreditCard, LogOut, Settings, User } from 'lucide-react'
import { fn } from 'storybook/test'

import { Button } from '@/shared/ui/button'

import { DropdownMenu } from './dropdown-menu'
import { DropdownMenuCheckboxItem } from './dropdown-menu-checkbox-item'
import { DropdownMenuContent } from './dropdown-menu-content'
import { DropdownMenuGroup } from './dropdown-menu-group'
import { DropdownMenuItem } from './dropdown-menu-item'
import { DropdownMenuLabel } from './dropdown-menu-label'
import { DropdownMenuRadioGroup } from './dropdown-menu-radio-group'
import { DropdownMenuRadioItem } from './dropdown-menu-radio-item'
import { DropdownMenuSeparator } from './dropdown-menu-separator'
import { DropdownMenuShortcut } from './dropdown-menu-shortcut'
import { DropdownMenuSub } from './dropdown-menu-sub'
import { DropdownMenuSubContent } from './dropdown-menu-sub-content'
import { DropdownMenuSubTrigger } from './dropdown-menu-sub-trigger'
import { DropdownMenuTrigger } from './dropdown-menu-trigger'

type StoryArgs = {
    label: string
    showShortcuts: boolean
    disableItem: boolean
    onSelect: (value: string) => void
}

const meta: Meta<StoryArgs> = {
    title: 'Overlays/Dropdown Menu',
    argTypes: {
        label: {
            name: 'Trigger label',
            control: 'text',
            table: { category: 'Content' },
        },
        showShortcuts: {
            name: 'Show shortcuts',
            control: 'boolean',
            table: { category: 'Content' },
        },
        disableItem: {
            name: 'Disable an item',
            control: 'boolean',
            table: { category: 'State' },
        },
        onSelect: {
            table: { disable: true },
        },
    },
    args: {
        label: 'Open menu',
        showShortcuts: true,
        disableItem: false,
        onSelect: fn(),
    },
    render: ({ label, showShortcuts, disableItem, onSelect }) => {
        const Story = () => {
            const [showStatus, setShowStatus] = useState(true)
            const [position, setPosition] = useState('top')

            return (
                <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                        <Button variant="outline">{label}</Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="start" className="w-56">
                        <DropdownMenuLabel>My Account</DropdownMenuLabel>
                        <DropdownMenuSeparator />
                        <DropdownMenuGroup>
                            <DropdownMenuItem onSelect={() => onSelect('profile')}>
                                <User />
                                Profile
                                {showShortcuts && <DropdownMenuShortcut>⇧⌘P</DropdownMenuShortcut>}
                            </DropdownMenuItem>
                            <DropdownMenuItem onSelect={() => onSelect('billing')}>
                                <CreditCard />
                                Billing
                                {showShortcuts && <DropdownMenuShortcut>⌘B</DropdownMenuShortcut>}
                            </DropdownMenuItem>
                            <DropdownMenuItem disabled={disableItem} onSelect={() => onSelect('settings')}>
                                <Settings />
                                Settings
                                {showShortcuts && <DropdownMenuShortcut>⌘S</DropdownMenuShortcut>}
                            </DropdownMenuItem>
                        </DropdownMenuGroup>
                        <DropdownMenuSeparator />
                        <DropdownMenuCheckboxItem checked={showStatus} onCheckedChange={setShowStatus}>
                            Show status bar
                        </DropdownMenuCheckboxItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuLabel inset>Panel position</DropdownMenuLabel>
                        <DropdownMenuRadioGroup value={position} onValueChange={setPosition}>
                            <DropdownMenuRadioItem value="top">Top</DropdownMenuRadioItem>
                            <DropdownMenuRadioItem value="bottom">Bottom</DropdownMenuRadioItem>
                            <DropdownMenuRadioItem value="right">Right</DropdownMenuRadioItem>
                        </DropdownMenuRadioGroup>
                        <DropdownMenuSeparator />
                        <DropdownMenuSub>
                            <DropdownMenuSubTrigger>More options</DropdownMenuSubTrigger>
                            <DropdownMenuSubContent>
                                <DropdownMenuItem onSelect={() => onSelect('invite')}>Invite users</DropdownMenuItem>
                                <DropdownMenuItem onSelect={() => onSelect('team')}>New team</DropdownMenuItem>
                            </DropdownMenuSubContent>
                        </DropdownMenuSub>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem variant="destructive" onSelect={() => onSelect('logout')}>
                            <LogOut />
                            Log out
                            {showShortcuts && <DropdownMenuShortcut>⇧⌘Q</DropdownMenuShortcut>}
                        </DropdownMenuItem>
                    </DropdownMenuContent>
                </DropdownMenu>
            )
        }

        return <Story />
    },
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'DropdownMenu',
}
