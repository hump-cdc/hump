import type { PromMetrics, PromSample } from '../types'

/**
 * Tracked names bucketed by length, so the ~70% of a scrape that is Go and process runtime
 * metrics is rejected on a number lookup without slicing its name out first.
 */
const indexByLength = (tracked: readonly string[]): Map<number, string[]> => {
    const byLength = new Map<number, string[]>()

    for (const name of tracked) {
        const bucket = byLength.get(name.length)
        if (bucket) bucket.push(name)
        else byLength.set(name.length, [name])
    }

    return byLength
}

const HASH = 0x23 // #
const QUOTE = 0x22 // "
const COMMA = 0x2c // ,
const BACKSLASH = 0x5c // \
const BRACE_OPEN = 0x7b // {
const BRACE_CLOSE = 0x7d // }

// Anything at or below the space separates, which covers tabs and a CRLF scrape's return.
const SPACE = 0x20

/** Resolve the exposition format's label-value escapes (`\\`, `\"`, `\n`). */
const unescapeLabel = (value: string): string =>
    value.replace(/\\(.)/g, (_, char: string) => (char === 'n' ? '\n' : char))

/**
 * Read the label list at `start` (the index of `{`), returning it and the index past the
 * closing `}`, or `-1` for a line too malformed to place a value on. Every search is bounded
 * by `stop`, or a truncated final line would read on into the rest of the document.
 */
const readLabels = (text: string, start: number, stop: number): { labels: Record<string, string>; end: number } => {
    const labels: Record<string, string> = {}

    let index = start + 1
    while (index < stop) {
        while (index < stop && text.charCodeAt(index) <= SPACE) index += 1
        if (index >= stop || text.charCodeAt(index) === BRACE_CLOSE) break

        // The label name runs to `=`, the value is quoted. Anything else is malformed.
        const equals = text.indexOf('=', index)
        if (equals < 0 || equals >= stop) return { labels, end: -1 }

        const open = text.indexOf('"', equals)
        if (open < 0 || open >= stop) return { labels, end: -1 }

        // Walk to the closing quote over any backslash escapes; only a value that had one
        // needs the unescape regex.
        let close = open + 1
        let escaped = false
        while (close < stop && text.charCodeAt(close) !== QUOTE) {
            if (text.charCodeAt(close) === BACKSLASH) {
                escaped = true
                close += 2
            } else close += 1
        }
        if (close >= stop) return { labels, end: -1 }

        let nameEnd = equals
        while (nameEnd > index && text.charCodeAt(nameEnd - 1) <= SPACE) nameEnd -= 1

        const value = text.slice(open + 1, close)
        labels[text.slice(index, nameEnd)] = escaped ? unescapeLabel(value) : value

        // Skip to the separator before consuming it: a space ahead of the comma would
        // otherwise leave it to be read as the head of the next label's name.
        index = close + 1
        while (index < stop && text.charCodeAt(index) <= SPACE) index += 1
        if (index < stop && text.charCodeAt(index) === COMMA) index += 1
    }

    // `}` is only looked for at a label-name boundary, so one inside a quoted value does not
    // end the list — the `key` label carries a user template and can hold any of `},="\`.
    return { labels, end: index + 1 }
}

/** Parses one Prometheus text exposition into the families its factory was given. */
export type MetricsParser = (text: string) => PromMetrics

/**
 * Build a parser over `tracked`, which owns the length index the scan rejects untracked
 * families with — so the allowlist is visible at the call site instead of baked in here, and
 * the parser stays free of any knowledge of what the app happens to collect.
 *
 * The returned parser skips comments, untracked families and non-finite values. Its scan walks
 * `text` by index and slices nothing until a line has survived the filter, so the discarded
 * majority of a scrape costs no allocation.
 */
export const createParser = (tracked: readonly string[]): MetricsParser => {
    const trackedByLength = indexByLength(tracked)

    return (text: string): PromMetrics => {
        const metrics: PromMetrics = new Map()
        const length = text.length

        let at = 0
        while (at < length) {
            const newline = text.indexOf('\n', at)
            const lineEnd = newline < 0 ? length : newline

            // Trim by moving the bounds inwards rather than allocating a trimmed line.
            let begin = at
            let stop = lineEnd
            while (stop > begin && text.charCodeAt(stop - 1) <= SPACE) stop -= 1
            while (begin < stop && text.charCodeAt(begin) <= SPACE) begin += 1

            at = lineEnd + 1

            if (begin === stop || text.charCodeAt(begin) === HASH) continue

            // The name ends at whichever comes first: the label list or the value separator.
            let nameEnd = begin
            while (nameEnd < stop) {
                const code = text.charCodeAt(nameEnd)
                if (code === BRACE_OPEN || code <= SPACE) break
                nameEnd += 1
            }
            if (nameEnd === stop) continue

            const candidates = trackedByLength.get(nameEnd - begin)
            if (candidates === undefined) continue

            // The match is the tracked constant itself, so a family's samples all share one string.
            let name = ''
            for (const candidate of candidates) {
                if (text.startsWith(candidate, begin)) {
                    name = candidate
                    break
                }
            }
            if (name === '') continue

            let labels: Record<string, string> = {}
            let after = nameEnd
            if (text.charCodeAt(nameEnd) === BRACE_OPEN) {
                const read = readLabels(text, nameEnd, stop)
                if (read.end < 0) continue

                labels = read.labels
                after = read.end
            }

            // The value ends at the first separator, which drops the optional trailing timestamp
            // the format allows after it.
            let valueStart = after
            while (valueStart < stop && text.charCodeAt(valueStart) <= SPACE) valueStart += 1

            let valueEnd = valueStart
            while (valueEnd < stop && text.charCodeAt(valueEnd) > SPACE) valueEnd += 1
            if (valueStart === valueEnd) continue

            const value = Number.parseFloat(text.slice(valueStart, valueEnd))
            if (!Number.isFinite(value)) continue

            const sample: PromSample = { name, labels, value }
            const samples = metrics.get(name)
            if (samples) samples.push(sample)
            else metrics.set(name, [sample])
        }

        return metrics
    }
}
