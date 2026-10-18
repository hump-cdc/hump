import { fuzzyFilter } from 'prism-react-editor/autocomplete'

import { FUNCTIONS, KEYWORDS, LITERALS, WORD_OPERATORS } from './where-input.grammar'

/** A column the condition can reference. */
export type WhereField = {
    name: string
    /** Column type as PostgreSQL formats it, shown next to the name. */
    type?: string
}

export type WhereSuggestionKind = 'field' | 'function' | 'keyword' | 'operator'

export type WhereSuggestion = {
    label: string
    kind: WhereSuggestionKind
    detail?: string
    /** Text replacing the completed word. */
    insert: string
    /** Caret offset within `insert` after insertion. */
    caret: number
    /** Matched label ranges as [start, end, start, end, …]. */
    matches: number[]
}

export type WhereCompletion = {
    /** Range of the document replaced by the chosen suggestion. */
    from: number
    to: number
    options: WhereSuggestion[]
}

type Candidate = Omit<WhereSuggestion, 'matches'>

const MAX_OPTIONS = 50

const KIND_ORDER: Record<WhereSuggestionKind, number> = { field: 0, operator: 1, function: 2, keyword: 3 }

const candidate = (
    label: string,
    kind: WhereSuggestionKind,
    insert = label,
    caret = insert.length,
    detail?: string,
) => ({
    label,
    kind,
    insert,
    caret,
    detail,
})

const OPERAND_KEYWORDS: Candidate[] = [
    ...['NOT', 'EXISTS', 'CASE', 'INTERVAL', 'CURRENT_DATE', 'CURRENT_TIMESTAMP', 'CURRENT_USER'].map((label) =>
        candidate(label, 'keyword'),
    ),
    ...LITERALS.filter((literal) => literal !== 'unknown').map((literal) =>
        candidate(literal.toUpperCase(), 'keyword'),
    ),
]

const FUNCTION_CANDIDATES: Candidate[] = FUNCTIONS.map((name) =>
    candidate(name, 'function', `${name}()`, name.length + 1),
)

const OPERATOR_CANDIDATES: Candidate[] = [
    candidate('AND', 'operator'),
    candidate('OR', 'operator'),
    candidate('LIKE', 'operator'),
    candidate('ILIKE', 'operator'),
    candidate('NOT LIKE', 'operator'),
    candidate('NOT ILIKE', 'operator'),
    candidate('SIMILAR TO', 'operator'),
    candidate('NOT SIMILAR TO', 'operator'),
    candidate('IN (…)', 'operator', 'IN ()', 4),
    candidate('NOT IN (…)', 'operator', 'NOT IN ()', 8),
    candidate('BETWEEN … AND …', 'operator', 'BETWEEN  AND ', 8),
    candidate('IS NULL', 'operator'),
    candidate('IS NOT NULL', 'operator'),
    candidate('IS TRUE', 'operator'),
    candidate('IS FALSE', 'operator'),
    candidate('IS DISTINCT FROM', 'operator'),
    candidate('IS NOT DISTINCT FROM', 'operator'),
]

const SYMBOL_OPERATORS: Candidate[] = ['=', '<>', '<', '<=', '>', '>=', '~', '~*'].map((label) =>
    candidate(label, 'operator'),
)

const AFTER_IS: Candidate[] = ['NULL', 'NOT NULL', 'TRUE', 'FALSE', 'DISTINCT FROM', 'NOT DISTINCT FROM'].map((label) =>
    candidate(label, 'keyword'),
)

const AFTER_IS_NOT: Candidate[] = ['NULL', 'TRUE', 'FALSE', 'DISTINCT FROM'].map((label) => candidate(label, 'keyword'))

const RESERVED = new Set([...WORD_OPERATORS, ...KEYWORDS, ...LITERALS])

/** Lexical state at the end of `text`: completions are offered in plain code only. */
const isInCode = (text: string) => {
    let i = 0
    while (i < text.length) {
        const ch = text[i]
        const next = text[i + 1]
        if (ch === '-' && next === '-') {
            const end = text.indexOf('\n', i)
            if (end < 0) return false
            i = end + 1
        } else if (ch === '/' && next === '*') {
            const end = text.indexOf('*/', i + 2)
            if (end < 0) return false
            i = end + 2
        } else if (ch === "'" || ch === '"') {
            const end = text.indexOf(ch, i + 1)
            if (end < 0) return false
            i = end + 1
        } else if (ch === '$') {
            const tag = /^\$\w*\$/.exec(text.slice(i))?.[0]
            if (!tag) {
                i++
                continue
            }
            const end = text.indexOf(tag, i + tag.length)
            if (end < 0) return false
            i = end + tag.length
        } else {
            i++
        }
    }
    return true
}

/** Quotes a column name unless it is a plain lower-case identifier. */
const quoteIdent = (name: string) =>
    /^[a-z_][a-z0-9_$]*$/.test(name) && !RESERVED.has(name) ? name : `"${name.replaceAll('"', '""')}"`

/** Whether the text right before the completed word ends an operand (column, literal, `)`). */
const endsWithOperand = (text: string) => {
    if (/[)'"\]]$|\b\d+(?:\.\d*)?$/.test(text)) return true
    const word = /([a-z_][\w$]*)$/i.exec(text)?.[1]?.toLowerCase()
    if (!word) return false
    return !WORD_OPERATORS.includes(word) && !KEYWORDS.includes(word)
}

const candidatesAt = (prev: string, fields: WhereField[], hasPrefix: boolean): Candidate[] => {
    if (/\bis\s+not$/i.test(prev)) return AFTER_IS_NOT
    if (/\bis$/i.test(prev)) return AFTER_IS
    if (endsWithOperand(prev)) return hasPrefix ? OPERATOR_CANDIDATES : [...SYMBOL_OPERATORS, ...OPERATOR_CANDIDATES]

    const fieldCandidates = fields.map((field) => {
        const insert = quoteIdent(field.name)
        return candidate(field.name, 'field', insert, insert.length, field.type)
    })
    return [...fieldCandidates, ...FUNCTION_CANDIDATES, ...OPERAND_KEYWORDS]
}

/**
 * Suggestions for the word at `pos` in a `WHERE` condition, or `null` when none apply.
 * Implicit queries (while typing) need a word prefix; explicit ones (Ctrl+Space) also complete at a blank.
 */
export const getCompletions = (
    value: string,
    pos: number,
    fields: WhereField[],
    explicit: boolean,
): WhereCompletion | null => {
    const before = value.slice(0, pos)
    if (!isInCode(before)) return null

    const prefix = /[a-z_][\w$]*$/i.exec(before)?.[0] ?? ''
    if (!prefix && !explicit) return null

    const from = pos - prefix.length
    const charBefore = before[from - 1]
    // Qualified names, casts and numbers like `1e5` aren't completed.
    if (charBefore === '.' || charBefore === ':' || /\d/.test(charBefore ?? '')) return null

    const to = pos + (/^[\w$]*/.exec(value.slice(pos))?.[0].length ?? 0)
    const prev = before.slice(0, from).trimEnd()

    const options = candidatesAt(prev, fields, prefix.length > 0)
        .flatMap((option) => {
            const match = fuzzyFilter(prefix, option.label)
            return match ? [{ option, score: match[0], matches: match[1] }] : []
        })
        // Stable sort: equal matches keep their declared order (fields in table order).
        .sort((a, b) => b.score - a.score || KIND_ORDER[a.option.kind] - KIND_ORDER[b.option.kind])
        .slice(0, MAX_OPTIONS)
        .map(({ option, matches }) => ({ ...option, matches }))

    if (options.length === 0) return null
    // The word is already typed out in full: nothing left to suggest.
    if (!explicit && options.length === 1 && options[0].insert.toLowerCase() === prefix.toLowerCase()) return null

    return { from, to, options }
}
