import type { FC } from 'react'

import { ChevronsRight } from 'lucide-react'

import { LiveControls } from '@/entities/config/state'
import { Header, HeaderTitle } from '@/shared/ui/header'
import { SidebarTrigger, useSidebar } from '@/shared/ui/sidebar'
import { useStaticData } from '@/shared/utils'

import { ConnectionInfo } from './connection-info'

export const AppHeader: FC = () => {
    const { title, headerAside: aside } = useStaticData()
    const { isMobile, state } = useSidebar()

    return (
        <Header>
            <div className="flex min-w-0 flex-nowrap items-center gap-2.5">
                <SidebarTrigger className="lg:hidden" />
                {!isMobile && state === 'collapsed' && (
                    <SidebarTrigger className="hidden lg:inline-flex">
                        <ChevronsRight />
                    </SidebarTrigger>
                )}
                <HeaderTitle>{title ?? ''}</HeaderTitle>

                <div className="hidden flex-nowrap items-center gap-2.5 lg:flex">
                    <span className="font-medium text-ink-tertiary">·</span>
                    <ConnectionInfo />
                </div>
            </div>
            <div className="flex flex-nowrap items-center gap-2">
                {aside}

                <LiveControls />
            </div>
        </Header>
    )
}
