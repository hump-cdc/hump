import type { MetricFamily } from '../metrics'
import { keyOf } from '../parse/series-key'
import type { LabelMatch, PromMetrics, PromSample, PromSeries } from '../types'

/** The family's samples, optionally narrowed to those matching `match`. */
export const samplesOf = (metrics: PromMetrics, name: MetricFamily, match?: LabelMatch): PromSample[] => {
    const samples = metrics.get(name) ?? []
    return match ? samples.filter(({ labels }) => match(labels)) : samples
}

/** Value of an unlabeled single-sample family; `undefined` when it is absent. */
export const gauge = (metrics: PromMetrics, name: MetricFamily): number | undefined => metrics.get(name)?.[0]?.value

/** Sum of the family's matching samples; `0` when the family is absent from the scrape. */
export const sumOf = (metrics: PromMetrics, name: MetricFamily, match?: LabelMatch): number =>
    samplesOf(metrics, name, match).reduce((total, { value }) => total + value, 0)

/** One entry per matching label set, keyed by its canonical identity. */
export const seriesOf = (metrics: PromMetrics, name: MetricFamily, match?: LabelMatch): PromSeries[] =>
    samplesOf(metrics, name, match).map((sample) => ({ key: keyOf(sample), value: sample.value }))
