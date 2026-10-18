import { describe, expect, test } from 'bun:test'

import { formatTime } from './format.time'

// Dates are built from local components, so the local-time output is the same in every time zone.

describe('formatTime', () => {
    test('prints local 24-hour time with milliseconds', () => {
        expect(formatTime(new Date(2026, 0, 2, 7, 5, 9, 42))).toBe('07:05:09.042')
        expect(formatTime(new Date(2026, 0, 2, 18, 30, 0, 999))).toBe('18:30:00.999')
    })
})
