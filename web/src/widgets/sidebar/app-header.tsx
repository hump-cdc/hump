import type { FC } from 'react'

import { ChevronsRight } from 'lucide-react'

import { Badge } from '@/shared/ui/badge'
import { Header, HeaderTitle } from '@/shared/ui/header'
import { SidebarTrigger, useSidebar } from '@/shared/ui/sidebar'
import { useStaticData } from '@/shared/utils'

import { ConnectionInfo } from './connection-info'

export const AppHeader: FC = () => {
    const { title, headerAside: aside } = useStaticData()
    const { isMobile, state } = useSidebar()

    return (
        <Header>
            <div className="flex flex-nowrap min-w-0 items-center gap-2.5">
                <SidebarTrigger className="lg:hidden" />
                {!isMobile && state === 'collapsed' && (
                    <SidebarTrigger className="hidden lg:inline-flex">
                        <ChevronsRight />
                    </SidebarTrigger>
                )}
                <HeaderTitle>{title ?? ''}</HeaderTitle>

                <div className="lg:flex flex-nowrap items-center gap-2.5 hidden">
                    <span className="text-ink-tertiary font-medium">·</span>
                    <ConnectionInfo />
                </div>
            </div>
            <div className="flex flex-nowrap items-center gap-2">
                {aside}

                <Badge variant="healthy" dot>
                    live
                </Badge>
            </div>
        </Header>
    )
}
