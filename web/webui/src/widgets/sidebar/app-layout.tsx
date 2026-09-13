import type { FC, PropsWithChildren } from 'react'

import { SidebarInset, SidebarProvider } from '@/shared/ui/sidebar'

import { AppHeader } from './app-header'
import { AppSidebar } from './app-sidebar'

export const AppLayout: FC<PropsWithChildren> = ({ children }) => (
    <SidebarProvider>
        <AppSidebar />
        <SidebarInset>
            <AppHeader />
            {children}
        </SidebarInset>
    </SidebarProvider>
)
