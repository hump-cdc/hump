import { afterEach, beforeEach, describe, expect, type Mock, spyOn, test } from 'bun:test'

import { migrateTopologyMode, useTopologyModeStore } from './mode'

describe('migrateTopologyMode', () => {
    test('keeps a stored mode that still exists', () => {
        expect(migrateTopologyMode({ mode: 'summary' })).toEqual({ mode: 'summary' })
    })

    test('falls back to the default for an unknown mode', () => {
        expect(migrateTopologyMode({ mode: 'compact' })).toEqual({ mode: 'detailed' })
    })

    test('falls back to the default when nothing usable was stored', () => {
        expect(migrateTopologyMode(null)).toEqual({ mode: 'detailed' })
        expect(migrateTopologyMode({})).toEqual({ mode: 'detailed' })
    })
})

describe('useTopologyModeStore', () => {
    // Bun has no `localStorage`, so the persist middleware warns on every `set`; the warning is muted.
    let warn: Mock<typeof console.warn>

    beforeEach(() => {
        warn = spyOn(console, 'warn').mockImplementation(() => {})
        useTopologyModeStore.setState(useTopologyModeStore.getInitialState(), true)
    })

    afterEach(() => {
        warn.mockRestore()
    })

    test('starts in the detailed mode', () => {
        expect(useTopologyModeStore.getState().mode).toBe('detailed')
    })

    test('setMode switches the mode', () => {
        useTopologyModeStore.getState().setMode('summary')

        expect(useTopologyModeStore.getState().mode).toBe('summary')
    })
})
