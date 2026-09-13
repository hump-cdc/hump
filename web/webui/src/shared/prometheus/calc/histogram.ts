import type { HistogramBucket } from '../types'

/** Read a histogram's `le` label as a number; `+Inf` becomes `Infinity`, which sorts last. */
export const parseBound = (le: string): number => (le === '+Inf' ? Number.POSITIVE_INFINITY : Number.parseFloat(le))

/** Total observations a cumulative bucket list covers, i.e. the `+Inf` bucket. */
const totalOf = (buckets: HistogramBucket[]): number => buckets.at(-1)?.count ?? 0

/** Sum two cumulative bucket lists bound by bound, rolling a sink's targets into one distribution. */
export const mergeBuckets = (into: HistogramBucket[], from: HistogramBucket[]): HistogramBucket[] => {
    if (into.length === 0) return from.map((bucket) => ({ ...bucket }))

    const merged = new Map(into.map((bucket) => [bucket.le, bucket.count]))
    for (const { le, count } of from) merged.set(le, (merged.get(le) ?? 0) + count)

    return [...merged.entries()].map(([le, count]) => ({ le, count })).sort((a, b) => a.le - b.le)
}

/**
 * The value below which `q` of the observations fall, interpolated linearly within the
 * bucket — the same estimate as Prometheus' `histogram_quantile`. `undefined` when the
 * distribution is empty or the quantile lands in the unbounded `+Inf` bucket.
 */
export const quantile = (buckets: HistogramBucket[], q: number): number | undefined => {
    const total = totalOf(buckets)
    if (total <= 0) return undefined

    const rank = q * total

    let lowerBound = 0
    let lowerCount = 0
    for (const { le, count } of buckets) {
        if (count >= rank) {
            if (!Number.isFinite(le)) return undefined

            const withinBucket = count - lowerCount
            if (withinBucket <= 0) return lowerBound

            return lowerBound + ((le - lowerBound) * (rank - lowerCount)) / withinBucket
        }

        lowerBound = le
        lowerCount = count
    }

    return undefined
}
