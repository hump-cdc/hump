import type { FC } from 'react'

import { Camera, ChartLine, GitBranch, ListChecks, SlidersHorizontal } from 'lucide-react'

import { NavItem } from './nav-item'

export const AppSidebarMenu: FC = () => (
    <>
        <NavItem to="/" label="Flows" icon={GitBranch} />
        <NavItem to="/configuration" label="Configuration" icon={SlidersHorizontal} />
        <NavItem to="/operations" label="Operations" icon={ListChecks} />
        <NavItem to="/snapshot" label="Snapshot" icon={Camera} />
        <NavItem to="/statistics" label="Statistics" icon={ChartLine} />
    </>
)
