import { Link, type LinkProps, useMatchRoute } from '@tanstack/react-router'

import type { ComponentType, FC } from 'react'

import { SidebarMenuButton, SidebarMenuItem, useSidebar } from '@/shared/ui/sidebar'

type NavItemProps = {
    to: LinkProps['to']
    label: string
    icon: ComponentType
}

export const NavItem: FC<NavItemProps> = ({ to, label, icon: Icon }) => {
    const matchRoute = useMatchRoute()
    const isActive = Boolean(matchRoute({ to, fuzzy: to !== '/' }))
    const { isMobile, setOpenMobile } = useSidebar()

    return (
        <SidebarMenuItem>
            <SidebarMenuButton asChild isActive={isActive} tooltip={label}>
                <Link
                    to={to}
                    onClick={() => {
                        if (isMobile) setOpenMobile(false)
                    }}
                >
                    <Icon />
                    <span>{label}</span>
                </Link>
            </SidebarMenuButton>
        </SidebarMenuItem>
    )
}
