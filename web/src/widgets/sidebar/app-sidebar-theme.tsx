import type { FC } from 'react'

import { Monitor, Moon, Sun } from 'lucide-react'

import type { Theme } from '@/shared/theme'
import { useTheme } from '@/shared/theme'
import { SidebarMenuButton, SidebarMenuItem } from '@/shared/ui/sidebar'

const THEME_CYCLE: readonly Theme[] = ['light', 'dark']

const THEME_META: Record<Theme, { icon: typeof Monitor; label: string }> = {
    system: { icon: Monitor, label: 'System Theme' },
    light: { icon: Sun, label: 'Light Theme' },
    dark: { icon: Moon, label: 'Dark Theme' },
}

export const AppSidebarTheme: FC = () => {
    const { theme, setTheme } = useTheme()

    const { icon: Icon, label } = THEME_META[theme]
    const nextTheme = THEME_CYCLE[(THEME_CYCLE.indexOf(theme) + 1) % THEME_CYCLE.length]

    return (
        <SidebarMenuItem>
            <SidebarMenuButton tooltip={label} onClick={() => setTheme(nextTheme)}>
                <Icon />
                <span>{label}</span>
            </SidebarMenuButton>
        </SidebarMenuItem>
    )
}
