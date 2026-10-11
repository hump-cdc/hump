import type { Meta, StoryObj } from '@storybook/react'
import {
    BookOpen,
    ChevronsLeft,
    ExternalLink,
    GitBranch,
    House,
    ListChecks,
    Moon,
    Mountain,
    RefreshCw,
    SlidersHorizontal,
    Trash2,
} from 'lucide-react'

import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Header, HeaderTitle } from '@/shared/ui/header'

import { Sidebar } from './sidebar'
import { SidebarContent } from './sidebar-content'
import { SidebarProvider } from './sidebar-context'
import { SidebarFooter } from './sidebar-footer'
import { SidebarHeader } from './sidebar-header'
import { SidebarInset } from './sidebar-inset'
import { SidebarMenu } from './sidebar-menu'
import { SidebarMenuButton } from './sidebar-menu-button'
import { SidebarMenuItem } from './sidebar-menu-item'
import { SidebarSeparator } from './sidebar-separator'
import { SidebarTrigger } from './sidebar-trigger'

const navItems = [
    { key: 'home', label: 'Home', icon: House },
    { key: 'configuration', label: 'Configuration', icon: SlidersHorizontal },
    { key: 'flows', label: 'Flows', icon: GitBranch },
    { key: 'operations', label: 'Operations', icon: ListChecks },
] as const

type StoryArgs = {
    active: (typeof navItems)[number]['key']
    defaultOpen: boolean
}

const meta: Meta<StoryArgs> = {
    title: 'Layout/Sidebar',
    parameters: { layout: 'fullscreen', disableSideBySide: true, disablePadding: true },
    argTypes: {
        active: {
            name: 'Active item',
            control: 'select',
            options: navItems.map((item) => item.key),
            table: { category: 'State' },
        },
    },
    args: {
        active: 'home',
    },
    render: ({ active, defaultOpen }) => (
        <SidebarProvider defaultOpen={defaultOpen}>
            <Sidebar>
                <SidebarHeader>
                    <div className="flex items-center justify-between gap-2 group-data-[collapsible=icon]:justify-center">
                        <div className="flex min-w-0 items-center gap-2">
                            <span className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-surface-3 text-on-primary [&>svg]:size-4.5">
                                <Mountain />
                            </span>
                            <span className="flex items-center gap-1.5 font-semibold text-ink group-data-[collapsible=icon]:hidden">
                                hump
                                <ExternalLink className="size-3.5 text-ink-tertiary" />
                            </span>
                        </div>
                        <SidebarTrigger className="group-data-[collapsible=icon]:hidden">
                            <ChevronsLeft />
                        </SidebarTrigger>
                    </div>
                </SidebarHeader>

                <SidebarContent className="px-3">
                    <SidebarMenu>
                        {navItems.map(({ key, label, icon: Icon }) => (
                            <SidebarMenuItem key={key}>
                                <SidebarMenuButton isActive={key === active} tooltip={label}>
                                    <Icon />
                                    <span>{label}</span>
                                </SidebarMenuButton>
                            </SidebarMenuItem>
                        ))}
                    </SidebarMenu>
                </SidebarContent>

                <SidebarFooter>
                    <SidebarMenu>
                        <SidebarMenuItem>
                            <SidebarMenuButton tooltip="Clear data">
                                <Trash2 />
                                <span>Clear data</span>
                            </SidebarMenuButton>
                        </SidebarMenuItem>
                        <SidebarMenuItem>
                            <SidebarMenuButton tooltip="Theme">
                                <Moon />
                                <span>Theme</span>
                            </SidebarMenuButton>
                        </SidebarMenuItem>
                    </SidebarMenu>
                    <SidebarSeparator />
                    <SidebarMenu>
                        <SidebarMenuItem>
                            <SidebarMenuButton tooltip="Documentation">
                                <BookOpen />
                                <span>Documentation</span>
                            </SidebarMenuButton>
                        </SidebarMenuItem>
                    </SidebarMenu>
                    <p className="px-2 pt-1 text-center text-2xs text-ink-tertiary group-data-[collapsible=icon]:hidden">
                        v 1.0.0
                    </p>
                </SidebarFooter>
            </Sidebar>

            <SidebarInset>
                <Header>
                    <div className="flex min-w-0 items-center gap-2.5">
                        <SidebarTrigger className="lg:hidden" />
                        <HeaderTitle>Overview</HeaderTitle>
                        <Badge variant="active" border dot>
                            production
                        </Badge>
                    </div>
                    <Button variant="outline">
                        <RefreshCw />
                        Refresh
                    </Button>
                </Header>
                <div className="flex flex-1 flex-col gap-4 p-6">
                    <div className="h-24 rounded-xl border border-hairline bg-surface-1" />
                    <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                        <div className="h-32 rounded-xl border border-hairline bg-surface-1" />
                        <div className="h-32 rounded-xl border border-hairline bg-surface-1" />
                        <div className="h-32 rounded-xl border border-hairline bg-surface-1" />
                    </div>
                </div>
            </SidebarInset>
        </SidebarProvider>
    ),
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Sidebar',
}
