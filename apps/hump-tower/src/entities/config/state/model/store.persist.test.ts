import { afterAll, beforeAll, describe, expect, test } from 'bun:test'

import { cleanup, renderHook } from '@testing-library/react'

import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { importFresh } from '@/shared/testing/modules.fake'

// Loaded first so the fresh instance below is the one the coverage report keeps.
import { CHART_STEPS } from './store'

type StoreModule = typeof import('./store')

/** Key the store persists under. */
const STORAGE_KEY = 'hump.live-controls'

/** An instance created while the DOM is in place, so it persists to `localStorage`. */
let fresh: StoreModule

beforeAll(async () => {
    registerDom()
    localStorage.clear()
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ state: { chartStep: '10m' }, version: 0 }))
    fresh = await importFresh<StoreModule>(import.meta.resolve('./store'))
    await fresh.useLiveControlsStore.persist.rehydrate()
})

afterAll(async () => {
    cleanup()
    localStorage.clear()
    await unregisterDom()
})

describe('useLiveControlsStore persistence', () => {
    test('carries over a step stored by an older version', () => {
        expect(fresh.useLiveControlsStore.getState().chartStep).toBe('10m')
        expect(fresh.CHART_STEPS).toEqual(CHART_STEPS)
    })

    test('stores only the chart step', () => {
        fresh.useLiveControlsStore.getState().setChartStep('5m')

        expect(JSON.parse(localStorage.getItem(STORAGE_KEY) ?? 'null')).toEqual({
            state: { chartStep: '5m' },
            version: 1,
        })
    })

    test('reads the step through the hook', () => {
        const { result } = renderHook(() => fresh.useLiveControlsChartStep())

        expect(result.current).toBe('5m')
    })

    test('falls back to the default for anything it does not know', () => {
        expect(fresh.migrateLiveControls(null)).toEqual({ chartStep: '1m' })
        expect(fresh.migrateLiveControls({ chartStep: '30m' })).toEqual({ chartStep: '1m' })
    })
})
