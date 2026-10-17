import { describe, expect, test } from 'bun:test'

import { getCompletions } from './where-input.completions'

const FIELDS = [
    { name: 'id', type: 'bigint' },
    { name: 'status', type: 'text' },
    { name: 'created_at', type: 'timestamp with time zone' },
    { name: 'UpdatedAt', type: 'timestamp' },
]

const labels = (value: string, explicit = false, pos = value.length) =>
    getCompletions(value, pos, FIELDS, explicit)?.options.map((option) => option.label)

describe('getCompletions', () => {
    test('suggests fields for an operand prefix', () => {
        const result = getCompletions('sta', 3, FIELDS, false)
        expect(result?.from).toBe(0)
        expect(result?.to).toBe(3)
        expect(result?.options[0]).toMatchObject({ label: 'status', kind: 'field', detail: 'text' })
    })

    test('suggests operators after an operand', () => {
        expect(labels('status i')).toEqual(expect.arrayContaining(['ILIKE', 'IN (…)', 'IS NULL']))
        expect(labels("status = 'active' a")?.[0]).toBe('AND')
        expect(labels('status i')).not.toContain('id')
    })

    test('suggests fields after a word operator', () => {
        expect(labels('status is null and i')?.[0]).toBe('id')
    })

    test('suggests IS continuations', () => {
        expect(labels('status is n')).toEqual(expect.arrayContaining(['NULL', 'NOT NULL']))
    })

    test('quotes identifiers that need it and places the caret inside function calls', () => {
        const updated = getCompletions('upd', 3, FIELDS, false)?.options.find((option) => option.label === 'UpdatedAt')
        expect(updated?.insert).toBe('"UpdatedAt"')

        const lower = getCompletions('lowe', 4, FIELDS, false)?.options.find((option) => option.label === 'lower')
        expect(lower).toMatchObject({ insert: 'lower()', caret: 6 })
    })

    test('stays quiet inside strings, comments and quoted identifiers', () => {
        expect(labels("status = 'act")).toBeUndefined()
        expect(labels('-- sta')).toBeUndefined()
        expect(labels('/* sta')).toBeUndefined()
        expect(labels('"sta')).toBeUndefined()
    })

    test('needs a prefix unless invoked explicitly', () => {
        expect(labels('status = ')).toBeUndefined()
        expect(labels('status = ', true)).toEqual(expect.arrayContaining(['id', 'status']))
        expect(labels('status ', true)).toEqual(expect.arrayContaining(['=', 'LIKE']))
    })

    test('closes once the word is typed out', () => {
        expect(labels('status')).toBeUndefined()
    })

    test('replaces the whole word around the caret', () => {
        expect(getCompletions('stXtus', 2, FIELDS, false)).toMatchObject({ from: 0, to: 6 })
    })
})
