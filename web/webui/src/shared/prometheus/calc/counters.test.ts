import { describe, expect, test } from 'bun:test'

import { FAMILY, TRACKED_METRICS } from '../metrics'
import { createParser } from '../parse/parser'
import { seriesOf } from '../query/samples'
import type { PromSeries } from '../types'
import { counterWindow, readingsOf } from './counters'

const parse = createParser(TRACKED_METRICS)

const series = (...pairs: [string, number][]): PromSeries[] => pairs.map(([key, value]) => ({ key, value }))

describe('readingsOf', () => {
    test('collects every series of the named families, keyed by identity', () => {
        const metrics = parse(
            [
                `${FAMILY.sinkMessages}{host="a",status="success"} 10`,
                `${FAMILY.sinkMessages}{host="b",status="success"} 4`,
                `${FAMILY.commitLag} 1.5`,
            ].join('\n'),
        )

        expect(readingsOf(metrics, [FAMILY.sinkMessages])).toEqual({
            [`${FAMILY.sinkMessages}{host="a",status="success"}`]: 10,
            [`${FAMILY.sinkMessages}{host="b",status="success"}`]: 4,
        })
    })

    test('leaves out families it was not asked for', () => {
        const metrics = parse(`${FAMILY.commitLag} 1.5`)
        expect(readingsOf(metrics, [FAMILY.sinkMessages])).toEqual({})
    })
})

describe('counterWindow.delta', () => {
    test('sums the advance of every series past its baseline', () => {
        const window = counterWindow({ a: 10, b: 4 }, false)
        expect(window.delta(series(['a', 15], ['b', 9]))).toBe(10)
    })

    test('counts nothing for a series seen for the first time', () => {
        const window = counterWindow({ a: 10 }, false)
        expect(window.delta(series(['a', 12], ['fresh', 99]))).toBe(2)
    })

    test('takes the whole reading of every series after a declared restart', () => {
        const window = counterWindow({ a: 100 }, true)
        expect(window.delta(series(['a', 3]))).toBe(3)
    })

    test('takes the whole reading when a counter drops without a declared restart', () => {
        const window = counterWindow({ a: 100 }, false)
        expect(window.delta(series(['a', 3]))).toBe(3)
    })

    test('counts zero for a series that did not advance', () => {
        expect(counterWindow({ a: 10 }, false).delta(series(['a', 10]))).toBe(0)
    })

    test('is zero for an empty family', () => {
        expect(counterWindow({ a: 10 }, false).delta([])).toBe(0)
    })
})

describe('counterWindow.perSeries', () => {
    test('keeps the advances apart instead of summing them', () => {
        const window = counterWindow({ a: 10, b: 4 }, false)
        expect([...window.perSeries(series(['a', 15], ['b', 9]))]).toEqual([
            ['a', 5],
            ['b', 5],
        ])
    })

    test('omits a series with no baseline rather than recording 0', () => {
        const deltas = counterWindow({ a: 10 }, false).perSeries(series(['a', 10], ['fresh', 99]))

        expect(deltas.get('a')).toBe(0)
        expect(deltas.has('fresh')).toBe(false)
    })

    test('baselines every series at zero after a declared restart, including unseen ones', () => {
        const deltas = counterWindow({ a: 100 }, true).perSeries(series(['a', 3], ['fresh', 7]))

        expect(deltas.get('a')).toBe(3)
        expect(deltas.get('fresh')).toBe(7)
    })
})

describe('counterWindow over a scrape', () => {
    test('measures a poll against the snapshot the previous scrape primed', () => {
        const first = parse(`${FAMILY.sinkMessages}{host="a",status="success"} 10`)
        const second = parse(`${FAMILY.sinkMessages}{host="a",status="success"} 17`)

        const window = counterWindow(readingsOf(first, [FAMILY.sinkMessages]), false)
        expect(window.delta(seriesOf(second, FAMILY.sinkMessages))).toBe(7)
    })

    test('matches series across scrapes regardless of label order', () => {
        const first = parse(`${FAMILY.sinkMessages}{status="success",host="a"} 10`)
        const second = parse(`${FAMILY.sinkMessages}{host="a",status="success"} 17`)

        const window = counterWindow(readingsOf(first, [FAMILY.sinkMessages]), false)
        expect(window.delta(seriesOf(second, FAMILY.sinkMessages))).toBe(7)
    })
})
