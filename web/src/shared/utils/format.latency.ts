/** Milliseconds as a compact latency, e.g. 12 → "12ms", 1240 → "1.24s". */
export const formatLatency = (ms: number): string => {
    const value = Math.max(0, ms)

    if (value < 1000) return `${Math.round(value)}ms`
    return `${(value / 1000).toFixed(2)}s`
}
