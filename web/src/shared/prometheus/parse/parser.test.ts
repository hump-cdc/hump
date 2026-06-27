import { describe, expect, test } from 'bun:test'

import { FAMILY, TRACKED_METRICS } from '../metrics'
import { createParser } from './parser'

const parse = createParser(TRACKED_METRICS)

/** The single sample of `name`, asserting the family carried exactly one. */
const only = (text: string, name: string) => {
    const samples = parse(text).get(name)
    expect(samples).toHaveLength(1)
    return samples?.[0]
}

describe('family filtering', () => {
    test('keeps a tracked family', () => {
        expect(only(`${FAMILY.commitLag} 1.5`, FAMILY.commitLag)).toMatchObject({ labels: {}, value: 1.5 })
    })

    test('drops an untracked family', () => {
        expect(parse('go_gc_duration_seconds 0.5').size).toBe(0)
    })

    test('drops an untracked name of the same length as a tracked one', () => {
        const impostor = `${FAMILY.heap.slice(0, -1)}X`
        expect(impostor).toHaveLength(FAMILY.heap.length)
        expect(parse(`${impostor} 1`).size).toBe(0)
    })

    test('drops HELP and TYPE comments', () => {
        const text = [
            `# HELP ${FAMILY.commitLag} replication lag`,
            `# TYPE ${FAMILY.commitLag} gauge`,
            `${FAMILY.commitLag} 2`,
        ].join('\n')

        expect(parse(text).get(FAMILY.commitLag)).toHaveLength(1)
    })

    test('shares one name string across a family instead of slicing per sample', () => {
        const text = [`${FAMILY.asyncQueue}{q="a"} 1`, `${FAMILY.asyncQueue}{q="b"} 2`].join('\n')
        const [first, second] = parse(text).get(FAMILY.asyncQueue) ?? []

        expect(first?.name).toBe(FAMILY.asyncQueue)
        expect(first?.name).toBe(second?.name as string)
    })

    test('honours the allowlist it was built with, not a global one', () => {
        const narrow = createParser([FAMILY.heap])
        const text = [`${FAMILY.heap} 1`, `${FAMILY.commitLag} 2`].join('\n')

        expect([...narrow(text).keys()]).toEqual([FAMILY.heap])
    })
})

describe('labels', () => {
    test('reads several labels and preserves scrape order of samples', () => {
        const text = [
            `${FAMILY.sinkMessages}{type="kafka",host="a",status="success"} 10`,
            `${FAMILY.sinkMessages}{type="kafka",host="a",status="failure"} 3`,
        ].join('\n')

        expect(parse(text).get(FAMILY.sinkMessages)).toEqual([
            { name: FAMILY.sinkMessages, labels: { type: 'kafka', host: 'a', status: 'success' }, value: 10 },
            { name: FAMILY.sinkMessages, labels: { type: 'kafka', host: 'a', status: 'failure' }, value: 3 },
        ])
    })

    test('resolves the format escapes', () => {
        expect(only(`${FAMILY.asyncQueue}{key="a\\"b\\\\c\\nd"} 1`, FAMILY.asyncQueue)?.labels.key).toBe('a"b\\c\nd')
    })

    test('does not end the label list on a `}` inside a quoted value', () => {
        const sample = only(`${FAMILY.asyncQueue}{key="{{ .id }}",n="2"} 7`, FAMILY.asyncQueue)

        expect(sample?.labels).toEqual({ key: '{{ .id }}', n: '2' })
        expect(sample?.value).toBe(7)
    })

    test('tolerates spacing around the separators', () => {
        expect(only(`${FAMILY.asyncQueue}{ a = "1" , b = "2" } 5`, FAMILY.asyncQueue)?.labels).toEqual({
            a: '1',
            b: '2',
        })
    })

    test('reads an empty label list', () => {
        expect(only(`${FAMILY.asyncQueue}{} 4`, FAMILY.asyncQueue)?.labels).toEqual({})
    })
})

describe('values', () => {
    test('drops the optional trailing timestamp', () => {
        expect(only(`${FAMILY.commitLag} 1.25 1700000000000`, FAMILY.commitLag)?.value).toBe(1.25)
    })

    test.each([
        ['exponent', '1.5e3', 1500],
        ['negative', '-2', -2],
        ['zero', '0', 0],
    ])('parses a %s value', (_label, raw, expected) => {
        expect(only(`${FAMILY.commitLag} ${raw}`, FAMILY.commitLag)?.value).toBe(expected as number)
    })

    test.each(['NaN', '+Inf', '-Inf'])('drops the non-finite value %s', (raw) => {
        expect(parse(`${FAMILY.commitLag} ${raw}`).size).toBe(0)
    })

    test('drops a line carrying no value', () => {
        expect(parse(`${FAMILY.commitLag}`).size).toBe(0)
    })
})

describe('line handling', () => {
    test('trims a CRLF scrape', () => {
        expect(only(`${FAMILY.commitLag} 1\r\n`, FAMILY.commitLag)?.value).toBe(1)
    })

    test('reads a final line with no trailing newline', () => {
        expect(only(`${FAMILY.commitLag} 1`, FAMILY.commitLag)?.value).toBe(1)
    })

    test('skips blank lines', () => {
        expect(parse(`\n\n${FAMILY.commitLag} 1\n\n`).get(FAMILY.commitLag)).toHaveLength(1)
    })

    test('drops a truncated label list without reading into the next line', () => {
        const text = [`${FAMILY.asyncQueue}{q="a`, `${FAMILY.commitLag} 9`].join('\n')
        const metrics = parse(text)

        expect(metrics.has(FAMILY.asyncQueue)).toBe(false)
        expect(metrics.get(FAMILY.commitLag)?.[0]?.value).toBe(9)
    })

    test('drops a label list with no closing brace at end of input', () => {
        expect(parse(`${FAMILY.asyncQueue}{q="a"} `).size).toBe(0)
    })

    test('returns an empty map for an empty scrape', () => {
        expect(parse('').size).toBe(0)
    })
})
