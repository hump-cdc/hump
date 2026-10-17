import { describe, expect, test } from 'bun:test'

import { migrateTopologyMode } from './mode'

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
