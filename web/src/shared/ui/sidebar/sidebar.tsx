import type { ComponentProps, CSSProperties, FC } from 'react'

import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '@/shared/ui/sheet'
import { cn } from '@/shared/utils'

import { SIDEBAR_WIDTH_MOBILE } from './constants'
import { useSidebar } from './sidebar-context'
import { SidebarRail } from './sidebar-rail'

type SidebarProps = ComponentProps<'div'> & {
    side?: 'left' | 'right'
}

export const Sidebar: FC<SidebarProps> = ({ side = 'left', className, children, ...props }) => {
    const { isMobile, state, openMobile, setOpenMobile } = useSidebar()

    if (isMobile) {
        return (
            <Sheet open={openMobile} onOpenChange={setOpenMobile} {...props}>
                <SheetContent
                    data-slot="sidebar"
                    data-mobile="true"
                    side={side}
                    showCloseButton={false}
                    className="w-(--sidebar-width) select-none bg-canvas p-0 text-ink"
                    style={{ '--sidebar-width': SIDEBAR_WIDTH_MOBILE } as CSSProperties}
                >
                    <SheetHeader className="sr-only">
                        <SheetTitle>Sidebar</SheetTitle>
                        <SheetDescription>Displays the mobile sidebar navigation.</SheetDescription>
                    </SheetHeader>
                    <div className="flex h-full w-full flex-col">{children}</div>
                </SheetContent>
            </Sheet>
        )
    }

    return (
        <div
            className="group peer hidden select-none text-ink lg:block"
            data-state={state}
            data-collapsible={state === 'collapsed' ? 'icon' : ''}
            data-side={side}
            data-slot="sidebar"
        >
            <div
                data-slot="sidebar-gap"
                className={cn(
                    'relative w-(--sidebar-width) bg-transparent transition-[width] duration-200 ease-linear',
                    'group-data-[collapsible=icon]:w-(--sidebar-width-icon)',
                )}
            />
            <div
                data-slot="sidebar-container"
                className={cn(
                    'fixed inset-y-0 z-10 hidden h-svh w-(--sidebar-width) transition-[left,right,width] duration-200 ease-linear lg:flex',
                    side === 'left' ? 'left-0' : 'right-0',
                    'group-data-[collapsible=icon]:w-(--sidebar-width-icon)',
                    'group-data-[side=left]:border-r group-data-[side=right]:border-l border-hairline',
                    className,
                )}
                {...props}
            >
                <div data-slot="sidebar-inner" className="flex h-full w-full flex-col bg-canvas">
                    {children}
                </div>
                <SidebarRail />
            </div>
        </div>
    )
}
