import { describe, expect, test } from 'bun:test'

import { clampHeight, collapsedPreview } from './where-input.helpers'

describe('clampHeight', () => {
    test('keeps room for the minimum rows', () => {
        expect(clampHeight(0)).toBe(76)
        expect(clampHeight(40)).toBe(76)
    })

    test('follows the content between the minimum and maximum rows', () => {
        expect(clampHeight(100)).toBe(102)
    })

    test('stops growing at the maximum rows', () => {
        expect(clampHeight(500)).toBe(216)
    })
})

describe('collapsedPreview', () => {
    test('is null for a blank value', () => {
        expect(collapsedPreview('')).toBeNull()
        expect(collapsedPreview(' \n\t\n ')).toBeNull()
    })

    test('highlights the first non-blank line', () => {
        const preview = collapsedPreview("\n  \nstatus = 'active'\n\n")

        expect(preview?.more).toBe(false)
        expect(preview?.html).toStartWith('status ')
        expect(preview?.html).toContain('<span class="token string">\'active\'</span>')
    })

    test('flags non-blank lines after the first one', () => {
        expect(collapsedPreview('id > 1\n  and status is null')).toMatchObject({
            html: expect.stringContaining('id'),
            more: true,
        })
    })
})
