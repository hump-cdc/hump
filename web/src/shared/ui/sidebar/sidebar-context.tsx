import type { ComponentProps, CSSProperties, FC } from 'react'
import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'

import { TooltipProvider } from '@/shared/ui/tooltip'
import { cn, useIsMobile } from '@/shared/utils'

import { SIDEBAR_KEYBOARD_SHORTCUT, SIDEBAR_STORAGE_KEY, SIDEBAR_WIDTH, SIDEBAR_WIDTH_ICON } from './constants'

type SidebarState = 'expanded' | 'collapsed'

type SidebarContextValue = {
    state: SidebarState
    open: boolean
    setOpen: (open: boolean) => void
    openMobile: boolean
    setOpenMobile: (open: boolean) => void
    isMobile: boolean
    toggleSidebar: () => void
}

const SidebarContext = createContext<SidebarContextValue | null>(null)

export const useSidebar = (): SidebarContextValue => {
    const context = useContext(SidebarContext)
    if (!context) throw new Error('useSidebar must be used within a SidebarProvider.')

    return context
}

const readStoredOpen = (fallback: boolean): boolean => {
    try {
        const stored = localStorage.getItem(SIDEBAR_STORAGE_KEY)
        if (stored === 'true') return true
        if (stored === 'false') return false
    } catch {
        // localStorage may be unavailable (private mode, blocked) — fall back to the default.
    }
    return fallback
}

type SidebarProviderProps = ComponentProps<'div'> & {
    defaultOpen?: boolean
    open?: boolean
    onOpenChange?: (open: boolean) => void
}

export const SidebarProvider: FC<SidebarProviderProps> = ({
    defaultOpen = true,
    open: openProp,
    onOpenChange: setOpenProp,
    className,
    style,
    children,
    ...props
}) => {
    const isMobile = useIsMobile()
    const [openMobile, setOpenMobile] = useState(false)

    const [internalOpen, setInternalOpen] = useState<boolean>(() => readStoredOpen(defaultOpen))
    const open = openProp ?? internalOpen

    const setOpen = useCallback(
        (value: boolean | ((value: boolean) => boolean)) => {
            const nextOpen = typeof value === 'function' ? value(open) : value
            if (setOpenProp) {
                setOpenProp(nextOpen)
            } else {
                setInternalOpen(nextOpen)
            }
            try {
                localStorage.setItem(SIDEBAR_STORAGE_KEY, String(nextOpen))
            } catch {
                // Ignore persistence failures — the in-memory choice still applies for this session.
            }
        },
        [open, setOpenProp],
    )

    const toggleSidebar = useCallback(() => {
        if (isMobile) {
            setOpenMobile((value) => !value)
        } else {
            setOpen((value) => !value)
        }
    }, [isMobile, setOpen])

    useEffect(() => {
        const onKeyDown = (event: KeyboardEvent) => {
            if (event.key === SIDEBAR_KEYBOARD_SHORTCUT && (event.metaKey || event.ctrlKey)) {
                event.preventDefault()
                toggleSidebar()
            }
        }
        window.addEventListener('keydown', onKeyDown)
        return () => window.removeEventListener('keydown', onKeyDown)
    }, [toggleSidebar])

    const state: SidebarState = open ? 'expanded' : 'collapsed'

    const mergedStyle = useMemo<CSSProperties>(
        () => ({
            '--sidebar-width': SIDEBAR_WIDTH,
            '--sidebar-width-icon': SIDEBAR_WIDTH_ICON,
            ...style,
        }),
        [style],
    )

    const value = useMemo<SidebarContextValue>(
        () => ({ state, open, setOpen, openMobile, setOpenMobile, isMobile, toggleSidebar }),
        [state, open, setOpen, openMobile, isMobile, toggleSidebar],
    )

    return (
        <SidebarContext.Provider value={value}>
            <TooltipProvider delayDuration={0}>
                <div
                    data-slot="sidebar-wrapper"
                    style={mergedStyle}
                    className={cn('group/sidebar-wrapper flex min-h-svh w-full', className)}
                    {...props}
                >
                    {children}
                </div>
            </TooltipProvider>
        </SidebarContext.Provider>
    )
}
