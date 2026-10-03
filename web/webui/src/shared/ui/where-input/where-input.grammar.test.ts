import { describe, expect, test } from 'bun:test'

import { languages, type Token, type TokenStream, tokenizeText } from 'prism-react-editor/prism'

import { WHERE_LANGUAGE } from './where-input.grammar'

/** Flattens top-level tokens into `[type, text]` pairs, skipping whitespace. */
const tokens = (code: string) =>
    tokenizeText(code, languages[WHERE_LANGUAGE])
        .filter((token): token is Token => typeof token !== 'string')
        .map((token) => [token.type, text(token.content)])

const text = (content: string | TokenStream): string =>
    typeof content === 'string'
        ? content
        : content.map((part) => (typeof part === 'string' ? part : text(part.content))).join('')

describe('pg-where grammar', () => {
    test('highlights comparisons, word operators and literals', () => {
        expect(tokens("status ILIKE 'act%' AND id >= 10 OR deleted_at IS NOT NULL")).toEqual([
            ['operator', 'ILIKE'],
            ['string', "'act%'"],
            ['operator', 'AND'],
            ['operator', '>='],
            ['number', '10'],
            ['operator', 'OR'],
            ['operator', 'IS'],
            ['operator', 'NOT'],
            ['boolean', 'NULL'],
        ])
    })

    test('highlights casts, quoted identifiers, escape strings and functions', () => {
        expect(tokens(`"CreatedAt"::timestamp with time zone > now() AND note = E'it''s'`)).toEqual([
            ['identifier', '"CreatedAt"'],
            ['cast', '::timestamp with time zone'],
            ['operator', '>'],
            ['function', 'now'],
            ['punctuation', '('],
            ['punctuation', ')'],
            ['operator', 'AND'],
            ['operator', '='],
            ['string', "E'it''s'"],
        ])
    })

    test('keeps keywords inside strings and comments plain', () => {
        expect(tokens("'a and b' -- or not")).toEqual([
            ['string', "'a and b'"],
            ['comment', '-- or not'],
        ])
    })
})
