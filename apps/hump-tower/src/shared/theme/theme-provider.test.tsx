import type { FC, PropsWithChildren } from 'react'

import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, spyOn, test } from 'bun:test'

import { act, cleanup, renderHook } from '@testing-library/react'

import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'

import { DARK_CLASS, THEME_STORAGE_KEY, THEME_TRANSITION_CLASS, THEME_TRANSITION_MS } from './constants'
import { ThemeProvider, useTheme } from './index'
// Type-only module: imported for its side effects alone so the coverage report lists it.
import './types'

/** The system color scheme the stubbed `matchMedia` reports, and its `change` listeners. */
const system = { isDark: false, listeners: new Set<() => void>() }

let originalMatchMedia: typeof window.matchMedia

beforeAll(() => {
    registerDom()
    originalMatchMedia = window.matchMedia
    Object.defineProperty(window, 'matchMedia', {
        configurable: true,
        writable: true,
        value: () => ({
            get matches() {
                return system.isDark
            },
            addEventListener: (_type: string, listener: () => void) => system.listeners.add(listener),
            removeEventListener: (_type: string, listener: () => void) => system.listeners.delete(listener),
        }),
    })
})
afterAll(async () => {
    window.matchMedia = originalMatchMedia
    await unregisterDom()
})
beforeEach(() => {
    localStorage.clear()
    system.isDark = false
    document.documentElement.className = ''
})
afterEach(cleanup)

const wrapper: FC<PropsWithChildren> = ({ children }) => <ThemeProvider>{children}</ThemeProvider>

describe('ThemeProvider', () => {
    test('starts on the default dark theme when nothing is stored', () => {
        const { result } = renderHook(() => useTheme(), { wrapper })

        expect(result.current.theme).toBe('dark')
        expect(result.current.resolvedTheme).toBe('dark')
    })

    test('restores the stored theme and resolves the stored system theme', () => {
        localStorage.setItem(THEME_STORAGE_KEY, 'light')
        expect(renderHook(() => useTheme(), { wrapper }).result.current.theme).toBe('light')

        localStorage.setItem(THEME_STORAGE_KEY, 'system')
        system.isDark = true
        expect(renderHook(() => useTheme(), { wrapper }).result.current).toMatchObject({
            theme: 'system',
            resolvedTheme: 'dark',
        })
    })

    test('falls back to the default theme when storage cannot be read', () => {
        const getItem = spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
            throw new Error('blocked')
        })

        try {
            expect(renderHook(() => useTheme(), { wrapper }).result.current.theme).toBe('dark')
        } finally {
            getItem.mockRestore()
        }
    })

    test('persists a chosen theme and animates the switch on <html>', async () => {
        const { result } = renderHook(() => useTheme(), { wrapper })
        const root = document.documentElement

        act(() => result.current.setTheme('light'))

        expect(result.current).toMatchObject({ theme: 'light', resolvedTheme: 'light' })
        expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('light')
        expect(root.classList.contains(DARK_CLASS)).toBe(false)
        expect(root.classList.contains(THEME_TRANSITION_CLASS)).toBe(true)

        await act(() => new Promise((resolve) => setTimeout(resolve, THEME_TRANSITION_MS + 20)))
        expect(root.classList.contains(THEME_TRANSITION_CLASS)).toBe(false)
    })

    test('still applies a chosen theme when storage cannot be written', () => {
        const setItem = spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
            throw new Error('quota')
        })

        try {
            const { result } = renderHook(() => useTheme(), { wrapper })
            act(() => result.current.setTheme('light'))
            expect(result.current.theme).toBe('light')
        } finally {
            setItem.mockRestore()
        }
    })

    test('follows the system scheme while on the system theme, and stops once left', () => {
        const { result } = renderHook(() => useTheme(), { wrapper })

        act(() => result.current.setTheme('system'))
        expect(result.current.resolvedTheme).toBe('light')

        system.isDark = true
        act(() => {
            for (const listener of system.listeners) listener()
        })
        expect(result.current.resolvedTheme).toBe('dark')
        expect(document.documentElement.classList.contains(DARK_CLASS)).toBe(true)

        act(() => result.current.setTheme('light'))
        expect(system.listeners.size).toBe(0)
    })
})

describe('useTheme', () => {
    test('throws outside a ThemeProvider', () => {
        const error = spyOn(console, 'error').mockImplementation(() => {})

        try {
            expect(() => renderHook(() => useTheme())).toThrow('useTheme must be used within a <ThemeProvider>')
        } finally {
            error.mockRestore()
        }
    })
})
