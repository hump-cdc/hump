import type { FC, PropsWithChildren } from 'react'
import { createContext, useCallback, useEffect, useState } from 'react'

import {
    DARK_CLASS,
    DARK_MEDIA_QUERY,
    DEFAULT_THEME,
    THEME_STORAGE_KEY,
    THEME_TRANSITION_CLASS,
    THEME_TRANSITION_MS,
} from './constants'
import type { ResolvedTheme, Theme } from './types'

export type ThemeContextValue = {
    theme: Theme
    resolvedTheme: ResolvedTheme
    setTheme: (theme: Theme) => void
}

export const ThemeContext = createContext<ThemeContextValue | null>(null)

const prefersDark = (): boolean => typeof window !== 'undefined' && window.matchMedia(DARK_MEDIA_QUERY).matches

const resolveTheme = (theme: Theme): ResolvedTheme => (theme === 'system' ? (prefersDark() ? 'dark' : 'light') : theme)

const readStoredTheme = (): Theme => {
    try {
        const stored = localStorage.getItem(THEME_STORAGE_KEY)
        if (stored === 'light' || stored === 'dark' || stored === 'system') return stored
    } catch {
        // localStorage may be unavailable (private mode, blocked) — fall back to default.
    }
    return DEFAULT_THEME
}

const applyResolvedTheme = (resolved: ResolvedTheme): void => {
    const root = document.documentElement
    root.classList.add(THEME_TRANSITION_CLASS)
    root.classList.toggle(DARK_CLASS, resolved === 'dark')
    window.setTimeout(() => {
        root.classList.remove(THEME_TRANSITION_CLASS)
    }, THEME_TRANSITION_MS)
}

export const ThemeProvider: FC<PropsWithChildren> = ({ children }) => {
    const [theme, setThemeState] = useState<Theme>(readStoredTheme)
    const [resolvedTheme, setResolvedTheme] = useState<ResolvedTheme>(() => resolveTheme(theme))

    const setTheme = useCallback((next: Theme) => {
        setThemeState(next)
        try {
            localStorage.setItem(THEME_STORAGE_KEY, next)
        } catch {
            // Ignore persistence failures — the in-memory choice still applies for this session.
        }
        const resolved = resolveTheme(next)
        setResolvedTheme(resolved)
        applyResolvedTheme(resolved)
    }, [])

    useEffect(() => {
        if (theme !== 'system') return

        const media = window.matchMedia(DARK_MEDIA_QUERY)
        const onChange = () => {
            const resolved = prefersDark() ? 'dark' : 'light'
            setResolvedTheme(resolved)
            applyResolvedTheme(resolved)
        }
        media.addEventListener('change', onChange)
        return () => media.removeEventListener('change', onChange)
    }, [theme])

    return <ThemeContext.Provider value={{ theme, resolvedTheme, setTheme }}>{children}</ThemeContext.Provider>
}
