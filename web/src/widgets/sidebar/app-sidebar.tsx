import type { FC } from 'react'

import { SiGithub as Github } from '@icons-pack/react-simple-icons'
import { BookOpen, ChevronsLeft, ExternalLink, Trash2 } from 'lucide-react'

import { Badge } from '@/shared/ui/badge'
import {
    Sidebar,
    SidebarContent,
    SidebarFooter,
    SidebarHeader,
    SidebarMenu,
    SidebarMenuButton,
    SidebarMenuItem,
    SidebarSeparator,
    SidebarTrigger,
} from '@/shared/ui/sidebar'

import { AppSidebarMenu } from './app-sidebar-menu'
import { AppSidebarTheme } from './app-sidebar-theme'
import { ConnectionInfo } from './connection-info'

export const AppSidebar: FC = () => {
    return (
        <Sidebar>
            <SidebarHeader>
                <div className="flex items-center justify-between gap-2 group-data-[collapsible=icon]:justify-center">
                    <div className="flex min-w-0 items-center gap-2">
                        <div className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-surface-3 text-on-primary">
                            <img src="/logo_sm.svg" alt="hump" className="size-5" />
                        </div>
                        <div className="flex items-center gap-1.5 font-semibold text-ink group-data-[collapsible=icon]:hidden">
                            hump
                            <Badge>{VERSION}</Badge>
                            <a href="https://hump.dev" target="_blank" rel="noreferrer">
                                <ExternalLink className="size-3.5 text-ink-tertiary" />
                            </a>
                        </div>
                    </div>
                    <SidebarTrigger className="group-data-[collapsible=icon]:hidden">
                        <ChevronsLeft />
                    </SidebarTrigger>
                </div>
            </SidebarHeader>

            <SidebarContent className="px-3 pt-1">
                <SidebarMenu>
                    <AppSidebarMenu />
                </SidebarMenu>

                <div className="flex lg:hidden flex-col gap-4 pt-4">
                    <h3 className="text-ink-tertiary font-semibold text-sm uppercase">Connection</h3>
                    <ConnectionInfo />
                </div>
            </SidebarContent>

            <SidebarFooter>
                <SidebarMenu>
                    <SidebarMenuItem>
                        <SidebarMenuButton tooltip="Clear data">
                            <Trash2 />
                            <span>Clear data</span>
                        </SidebarMenuButton>
                    </SidebarMenuItem>
                    <AppSidebarTheme />
                </SidebarMenu>
                <SidebarSeparator />
                <SidebarMenu>
                    <SidebarMenuItem>
                        <SidebarMenuButton tooltip="Github" asChild>
                            <a href="https://github.com/sergeyzwezdin/hump" target="_blank" rel="noreferrer">
                                <Github />
                                <span>
                                    Github
                                    <ExternalLink className="ml-2 mb-0.5 inline-block size-3.5 text-ink-tertiary" />
                                </span>
                            </a>
                        </SidebarMenuButton>
                    </SidebarMenuItem>
                    <SidebarMenuItem>
                        <SidebarMenuButton tooltip="Documentation" asChild>
                            <a href="https://hump.dev/docs" target="_blank" rel="noreferrer">
                                <BookOpen />
                                <span>
                                    Documentation
                                    <ExternalLink className="ml-2 mb-0.5 inline-block size-3.5 text-ink-tertiary" />
                                </span>
                            </a>
                        </SidebarMenuButton>
                    </SidebarMenuItem>
                </SidebarMenu>
            </SidebarFooter>
        </Sidebar>
    )
}
