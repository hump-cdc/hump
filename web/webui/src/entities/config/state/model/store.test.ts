import { describe, expect, test } from 'bun:test'

import { migrateLiveControls } from './store'

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
