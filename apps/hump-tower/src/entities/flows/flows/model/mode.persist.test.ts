import { afterAll, beforeAll, describe, expect, test } from 'bun:test'

import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { importFresh } from '@/shared/testing/modules.fake'

// Loaded first so the fresh instance below is the one the coverage report keeps.
import { TOPOLOGY_MODES } from './mode'

type ModeModule = typeof import('./mode')

/** Key the mode persists under. */
const STORAGE_KEY = 'hump.flows-mode'

/** An instance created while the DOM is in place, so it persists to `localStorage`. */
let fresh: ModeModule

beforeAll(async () => {
    registerDom()
    localStorage.clear()
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ state: { mode: 'summary' }, version: 0 }))
    fresh = await importFresh<ModeModule>(import.meta.resolve('./mode'))
    // Hydrated again explicitly, so the stored version-0 state goes through `migrate` before the tests.
    await fresh.useTopologyModeStore.persist.rehydrate()
})

afterAll(async () => {
    localStorage.clear()
    await unregisterDom()
})

describe('useTopologyModeStore persistence', () => {
    test('migrates a mode stored by an older version', () => {
        expect(fresh.TOPOLOGY_MODES).toEqual(TOPOLOGY_MODES)
        expect(fresh.useTopologyModeStore.getState().mode).toBe('summary')
    })

    test('stores the mode and nothing else', () => {
        fresh.useTopologyModeStore.getState().setMode('detailed')

        expect(JSON.parse(localStorage.getItem(STORAGE_KEY) ?? 'null')).toEqual({
            state: { mode: 'detailed' },
            version: 1,
        })
    })

    test('migrateTopologyMode falls back to the default for an unknown mode', () => {
        expect(fresh.migrateTopologyMode({ mode: 'compact' })).toEqual({ mode: 'detailed' })
    })
})
