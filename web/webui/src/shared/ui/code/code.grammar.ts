import { type Document, isCollection, parseDocument, visit } from 'yaml'

/** Grammars the code viewer can highlight with. */
export type CodeGrammar = 'yaml' | 'json' | 'sql' | 'plaintext'

/** Text to render and the grammar to highlight it with. */
export type CodeSource = { lang: CodeGrammar; text: string }

/** Parses `value` as JSON; `undefined` when it is not JSON. */
const parseJson = (value: string): { value: unknown } | undefined => {
    try {
        return { value: JSON.parse(value) }
    } catch {
        return undefined
    }
}

/** Parses `value` as a YAML map or list; `undefined` on errors or a bare scalar. */
const parseYaml = (value: string): Document | undefined => {
    const doc = parseDocument(value)
    return doc.errors.length === 0 && isCollection(doc.contents) ? doc : undefined
}

/**
 * Picks a grammar for `value`: JSON when it parses as JSON (checked first — JSON is also YAML),
 * YAML when it parses cleanly into a map or list, plain text otherwise (bare scalars included).
 */
export const detectGrammar = (value: string): CodeGrammar => {
    if (parseJson(value)) return 'json'
    if (parseYaml(value)) return 'yaml'
    return 'plaintext'
}

/**
 * Detects the grammar of `value` and re-emits JSON with a 2-space indent and YAML in block style
 * with a 2-space indent (comments kept). Plain text is returned as-is.
 */
export const formatCode = (value: string): CodeSource => {
    const json = parseJson(value)
    if (json) return { lang: 'json', text: JSON.stringify(json.value, null, 2) }

    const yaml = parseYaml(value)
    if (yaml) {
        visit(yaml, {
            Collection: (_, node) => {
                node.flow = false
            },
        })
        return { lang: 'yaml', text: yaml.toString({ indent: 2 }).trimEnd() }
    }

    return { lang: 'plaintext', text: value }
}
