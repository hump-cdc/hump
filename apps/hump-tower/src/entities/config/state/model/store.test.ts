import { afterEach, beforeEach, describe, expect, type Mock, spyOn, test } from 'bun:test'

import { CHART_STEPS, migrateLiveControls, useLiveControlsStore } from './store'

describe('migrateLiveControls', () => {
    test('keeps a stored step that is still offered', () => {
        expect(migrateLiveControls({ chartStep: '5m' })).toEqual({ chartStep: '5m' })
    })

    test('falls back to the default for an unknown step', () => {
        expect(migrateLiveControls({ chartStep: '30m' })).toEqual({ chartStep: '1m' })
    })

    test('falls back to the default when nothing usable was stored', () => {
        expect(migrateLiveControls(null)).toEqual({ chartStep: '1m' })
        expect(migrateLiveControls({})).toEqual({ chartStep: '1m' })
    })
})

describe('useLiveControlsStore', () => {
    // Bun has no `localStorage`, so the persist middleware warns on every `set`; the warning is muted.
    let warn: Mock<typeof console.warn>

    beforeEach(() => {
        warn = spyOn(console, 'warn').mockImplementation(() => {})
        useLiveControlsStore.setState(useLiveControlsStore.getInitialState(), true)
    })

    afterEach(() => {
        warn.mockRestore()
    })

    test('starts at the one-minute step', () => {
        expect(useLiveControlsStore.getState().chartStep).toBe('1m')
    })

    test('setChartStep selects another offered step', () => {
        useLiveControlsStore.getState().setChartStep('10m')

        expect(useLiveControlsStore.getState().chartStep).toBe('10m')
    })

    test('offers the steps in ascending order', () => {
        expect(CHART_STEPS).toEqual(['1m', '5m', '10m'])
    })
})
