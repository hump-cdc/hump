/** A single sample of a metric family: its label set and value. */
export type PromSample = {
    /** Owning family. The tracked constant itself, so it costs no per-sample allocation. */
    name: string
    labels: Record<string, string>
    value: number
    /** Canonical identity, filled in on first use by `keyOf`. */
    key?: string
}

/** One sample reduced to a stable identity and its value. */
export type PromSeries = { key: string; value: number }

/** Metric family name → its samples, in scrape order. Only tracked families are retained. */
export type PromMetrics = Map<string, PromSample[]>

/** Narrows a family to the samples whose labels match. */
export type LabelMatch = (labels: Record<string, string>) => boolean

/**
 * One bucket of a histogram: its upper bound and the count of observations at or below it.
 */
export type HistogramBucket = { le: number; count: number }

/** Last-seen cumulative reading of every tracked counter series, keyed by series identity. */
export type CounterSnapshot = Record<string, number>
