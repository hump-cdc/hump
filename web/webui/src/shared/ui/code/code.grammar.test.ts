import { describe, expect, test } from 'bun:test'

import { detectGrammar, formatCode } from './code.grammar'

describe('detectGrammar', () => {
    test('detects JSON, YAML and plain text', () => {
        expect(detectGrammar('{"a":1}')).toBe('json')
        expect(detectGrammar('a: 1')).toBe('yaml')
        expect(detectGrammar('hello')).toBe('plaintext')
    })
})

describe('formatCode', () => {
    test('pretty-prints JSON with a 2-space indent', () => {
        expect(formatCode('{"op":"update","key":{"id":1}}')).toEqual({
            lang: 'json',
            text: '{\n  "op": "update",\n  "key": {\n    "id": 1\n  }\n}',
        })
    })

    test('re-emits YAML in block style with a 2-space indent, keeping comments', () => {
        expect(formatCode('# sinks\nsinks:\n    - {name: a, url: x}')).toEqual({
            lang: 'yaml',
            text: '# sinks\nsinks:\n  - name: a\n    url: x',
        })
    })

    test('keeps plain text and truncated JSON as-is', () => {
        expect(formatCode('{"op":"upd')).toEqual({ lang: 'plaintext', text: '{"op":"upd' })
        expect(formatCode('hello')).toEqual({ lang: 'plaintext', text: 'hello' })
    })
})
