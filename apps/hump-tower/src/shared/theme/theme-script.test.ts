import { describe, expect, test } from 'bun:test'

import { DARK_CLASS, DARK_MEDIA_QUERY, THEME_STORAGE_KEY } from './constants'
import { themeInitScript } from './theme-script'

/** What the script touches, injected in place of the browser globals. */
type ScriptEnvironment = {
    stored: string | null
    systemDark: boolean
    failStorage?: boolean
}

/** Result of one run: the dark class toggle and the keys and queries the script read. */
type ScriptRun = {
    isDark: boolean | undefined
    readKeys: string[]
    queries: string[]
}

describe('themeInitScript', () => {
    test('applies a stored dark or light theme', () => {
        expect(runScript({ stored: 'dark', systemDark: false }).isDark).toBe(true)
        expect(runScript({ stored: 'light', systemDark: true }).isDark).toBe(false)
    })

    test('follows the system preference for the system theme', () => {
        expect(runScript({ stored: 'system', systemDark: true })).toEqual({
            isDark: true,
            readKeys: [THEME_STORAGE_KEY],
            queries: [DARK_MEDIA_QUERY],
        })
        expect(runScript({ stored: 'system', systemDark: false }).isDark).toBe(false)
    })

    test('falls back to the default dark theme when nothing valid is stored', () => {
        expect(runScript({ stored: null, systemDark: false }).isDark).toBe(true)
        expect(runScript({ stored: 'sepia', systemDark: false }).isDark).toBe(true)
    })

    test('leaves the page untouched when storage throws', () => {
        expect(runScript({ stored: null, systemDark: false, failStorage: true }).isDark).toBeUndefined()
    })
})

/** Evaluates the script against fake `localStorage`, `matchMedia` and `document`. */
const runScript = ({ stored, systemDark, failStorage = false }: ScriptEnvironment): ScriptRun => {
    const run: ScriptRun = { isDark: undefined, readKeys: [], queries: [] }

    const localStorage = {
        getItem: (key: string): string | null => {
            if (failStorage) throw new Error('storage is disabled')
            run.readKeys.push(key)
            return stored
        },
    }
    const matchMedia = (query: string): { matches: boolean } => {
        run.queries.push(query)
        return { matches: systemDark }
    }
    const document = {
        documentElement: {
            classList: {
                toggle: (name: string, force: boolean): void => {
                    expect(name).toBe(DARK_CLASS)
                    run.isDark = force
                },
            },
        },
    }

    new Function('localStorage', 'matchMedia', 'document', themeInitScript)(localStorage, matchMedia, document)

    return run
}
