import { describe, expect, test } from 'bun:test'

import { formatDateTime, formatDateTimeRange, formatTimeCompact } from './format.datetime'

// Dates are built from local components, so the local-time output is the same in every time zone.
// The separator between date and time is engine-dependent: ", " in Node, " at " in Bun.

describe('formatDateTime', () => {
    test('prints the day, short month, year and 24-hour time', () => {
        expect(formatDateTime(new Date(2026, 9, 17, 14, 3, 5))).toMatch(/^17 Oct 2026(, | at )14:03:05$/)
    })
})

describe('formatDateTimeRange', () => {
    test('prints only the end time when both ends fall on the same day', () => {
        const range = formatDateTimeRange(new Date(2026, 9, 17, 14, 3, 5), new Date(2026, 9, 17, 15, 0, 0))

        expect(range).toMatch(/^17 Oct 2026(, | at )14:03:05 → 15:00:00$/)
    })

    test('repeats the date when the end falls on another day', () => {
        const range = formatDateTimeRange(new Date(2026, 9, 17, 23, 59, 0), new Date(2026, 9, 18, 0, 1, 2))

        expect(range).toMatch(/^17 Oct 2026(, | at )23:59:00 → 18 Oct 2026(, | at )00:01:02$/)
    })
})

describe('formatTimeCompact', () => {
    test('prints zero-padded local hours and minutes', () => {
        expect(formatTimeCompact(new Date(2026, 0, 2, 7, 5).getTime())).toBe('07:05')
        expect(formatTimeCompact(new Date(2026, 0, 2, 23, 59).getTime())).toBe('23:59')
    })
})
