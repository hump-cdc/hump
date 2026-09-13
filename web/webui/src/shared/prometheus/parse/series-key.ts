import type { PromSample } from '../types'

/** True when a label value carries a character the exposition format escapes. */
const needsEscape = (value: string): boolean => value.includes('"') || value.includes('\\')

/**
 * Canonical `name{k="v",…}` identity of one sample, labels sorted and values re-escaped; the
 * persisted counter snapshot is keyed by it, so changing the rendering invalidates every
 * stored baseline. The plain sort only matches collation because every label name the
 * exporter emits is lowercase ASCII — see `internal/telemetry/metrics/metrics.go`.
 */
const seriesKey = (name: string, labels: Record<string, string>): string => {
    const names = Object.keys(labels)
    if (names.length === 0) return name
    if (names.length > 1) names.sort()

    let rendered = ''
    for (let index = 0; index < names.length; index += 1) {
        const key = names[index]
        const value = labels[key]

        if (index > 0) rendered += ','
        rendered += `${key}="${needsEscape(value) ? value.replace(/(["\\])/g, '\\$1') : value}"`
    }

    return `${name}{${rendered}}`
}

/**
 * {@link seriesKey} for a parsed sample, memoized on it — one scrape is keyed three times
 * over, for the baselines, the deltas and the flow graph's lookup of them.
 */
export const keyOf = (sample: PromSample): string => (sample.key ??= seriesKey(sample.name, sample.labels))
