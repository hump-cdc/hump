import type { FC } from 'react'

import { Camera, GitBranch, House, ListChecks, SlidersHorizontal } from 'lucide-react'

import { NavItem } from './nav-item'

export const AppSidebarMenu: FC = () => (
    <>
        <NavItem to="/" label="Home" icon={House} />
        <NavItem to="/configuration" label="Configuration" icon={SlidersHorizontal} />
        <NavItem to="/flows" label="Flows" icon={GitBranch} />
        <NavItem to="/operations" label="Operations" icon={ListChecks} />
        <NavItem to="/snapshot" label="Snapshot" icon={Camera} />
    </>
)
