import { languageMap } from 'prism-react-editor'
import { languages } from 'prism-react-editor/prism'

/** Prism language id of the PostgreSQL `WHERE` condition dialect. */
export const WHERE_LANGUAGE = 'pg-where'

/** Word operators, highlighted as operators and offered after an operand. */
export const WORD_OPERATORS = [
    'and',
    'or',
    'not',
    'in',
    'between',
    'symmetric',
    'like',
    'ilike',
    'similar',
    'to',
    'escape',
    'is',
    'distinct',
    'from',
]

/** Keywords that start or shape an operand. */
export const KEYWORDS = [
    'any',
    'all',
    'some',
    'exists',
    'case',
    'when',
    'then',
    'else',
    'end',
    'interval',
    'current_date',
    'current_time',
    'current_timestamp',
    'localtime',
    'localtimestamp',
    'current_user',
    'session_user',
]

/** Literal keywords. */
export const LITERALS = ['true', 'false', 'null', 'unknown']

/** Functions commonly used in row filters. */
export const FUNCTIONS = [
    'abs',
    'age',
    'array_length',
    'cardinality',
    'ceil',
    'char_length',
    'coalesce',
    'date_part',
    'date_trunc',
    'extract',
    'floor',
    'greatest',
    'jsonb_array_length',
    'jsonb_typeof',
    'least',
    'left',
    'length',
    'lower',
    'ltrim',
    'md5',
    'now',
    'nullif',
    'position',
    'regexp_like',
    'right',
    'round',
    'rtrim',
    'split_part',
    'starts_with',
    'substring',
    'to_char',
    'to_date',
    'to_timestamp',
    'trim',
    'upper',
]

const words = (list: string[]) => new RegExp(`\\b(?:${list.join('|')})\\b`, 'i')

languages[WHERE_LANGUAGE] = {
    comment: /--.*|\/\*[\s\S]*?\*\//g,
    // Standard, escape (E''), bit (B''/X'') and national (N'') strings, then dollar quoting.
    string: [/(?:\b[bnex])?'(?:[^']|'')*'/gi, /\$(\w*)\$[\s\S]*?\$\1\$/g],
    identifier: { pattern: /"(?:[^"]|"")*"/g, inside: { punctuation: /^"|"$/ } },
    cast: {
        pattern:
            /::\s*[a-z_][\w.]*(?:\s+(?:with(?:out)?\s+time\s+zone|varying|precision))?(?:\s*\(\s*\d+(?:\s*,\s*\d+)?\s*\))?(?:\[\])*/i,
        inside: { punctuation: /::|[()[\],]/ },
    },
    number: /\b\d+(?:\.\d*)?(?:e[+-]?\d+)?\b|\B\.\d+(?:e[+-]?\d+)?\b/i,
    boolean: words(LITERALS),
    keyword: words(KEYWORDS),
    operator: [
        { pattern: words(WORD_OPERATORS), alias: 'keyword' },
        /<>|!=|<=|>=|!~\*?|~\*?|@>|<@|&&|\?[|&]?|->>?|#>>?|\|\||[-=<>+*/%^#|&]/,
    ],
    function: /\b[a-z_]\w*(?=\s*\()/i,
    punctuation: /[()[\],.;]/,
}

languageMap[WHERE_LANGUAGE] = { comments: { line: '--', block: ['/*', '*/'] } }
