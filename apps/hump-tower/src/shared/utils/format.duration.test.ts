import { describe, expect, test } from 'bun:test'

import { formatDuration } from './format.duration'

describe('formatDuration', () => {
    test('prints seconds alone below a minute', () => {
        expect(formatDuration(0)).toBe('0s')
        expect(formatDuration(48)).toBe('48s')
    })

    test('prints minutes and seconds from one minute on', () => {
        expect(formatDuration(60)).toBe('1m 0s')
        expect(formatDuration(108)).toBe('1m 48s')
    })

    test('has no hours unit', () => {
        expect(formatDuration(3600)).toBe('60m 0s')
    })

    test('rounds to whole seconds', () => {
        expect(formatDuration(59.6)).toBe('1m 0s')
    })

    test('clamps a negative duration to zero', () => {
        expect(formatDuration(-3)).toBe('0s')
    })
})
