/** Milliseconds as a compact latency, e.g. 12 → "12ms", 1240 → "1.24s". */
export const formatLatency = (ms: number): string => {
    const value = Math.max(0, ms)

    if (value < 1000) return `${Math.round(value)}ms`
    return `${(value / 1000).toFixed(2)}s`
}

/** Milliseconds scaled to a compact latency unit, e.g. 12 → { value: 12, suffix: 'ms' }, 1240 → { value: 1.24, suffix: 's' }. */
export const scaleLatency = (ms: number): { value: number; suffix: string; fractionDigits: number } => {
    const value = Math.max(0, ms)

    if (value < 1000) return { value: Math.round(value), suffix: 'ms', fractionDigits: 0 }
    return { value: value / 1000, suffix: 's', fractionDigits: 2 }
}
