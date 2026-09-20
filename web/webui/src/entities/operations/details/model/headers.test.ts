import { describe, expect, test } from 'bun:test'

import { toHeaders } from './headers'

describe('toHeaders', () => {
    test('strips the header prefix and keeps other metadata as is, in order', () => {
        expect(toHeaders({ 'header.content-type': 'application/json', 'http.status': '200' })).toEqual([
            { key: 'content-type', value: 'application/json' },
            { key: 'http.status', value: '200' },
        ])
    })
})
