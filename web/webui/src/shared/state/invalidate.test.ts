import { QueryClient } from '@tanstack/react-query'

import { describe, expect, test } from 'bun:test'

import { invalidateForLiveEvent, invalidateForLiveReconnect } from './invalidate'
import { keys } from './keys'

type Call = { filters: unknown; options: unknown }

const spyClient = (): { client: QueryClient; calls: Call[] } => {
    const client = new QueryClient()
    const calls: Call[] = []

    client.invalidateQueries = ((filters: unknown, options: unknown) => {
        calls.push({ filters, options })
        return Promise.resolve()
    }) as QueryClient['invalidateQueries']

    return { client, calls }
}

describe('invalidateForLiveEvent', () => {
    test('invalidates the metrics prefix', () => {
        const { client, calls } = spyClient()
        invalidateForLiveEvent(client)

        expect(calls).toHaveLength(1)
        expect(calls[0].filters).toMatchObject({ queryKey: ['metrics'] })
    })

    test('never cancels an in-flight refetch', () => {
        const { client, calls } = spyClient()
        invalidateForLiveEvent(client)

        expect(calls[0].options).toEqual({ cancelRefetch: false })
    })
})

describe('the metrics prefix against real query keys', () => {
    test('invalidates every duration variant', () => {
        const client = new QueryClient()
        client.setQueryData(keys.queries.metrics('1m'), { points: [] })
        client.setQueryData(keys.queries.metrics('10m'), { points: [] })

        invalidateForLiveEvent(client)

        expect(client.getQueryState(keys.queries.metrics('1m'))?.isInvalidated).toBe(true)
        expect(client.getQueryState(keys.queries.metrics('10m'))?.isInvalidated).toBe(true)
    })

    test('leaves unrelated queries alone', () => {
        const client = new QueryClient()
        client.setQueryData(['config'], { version: '1' })

        invalidateForLiveEvent(client)

        expect(client.getQueryState(['config'])?.isInvalidated).toBe(false)
    })
})

describe('invalidateForLiveReconnect', () => {
    test('refetches inactive queries too, because events were missed', () => {
        const { client, calls } = spyClient()
        invalidateForLiveReconnect(client)

        expect(calls[0].filters).toMatchObject({ queryKey: ['metrics'], refetchType: 'all' })
        expect(calls[0].options).toEqual({ cancelRefetch: false })
    })

    test('marks operations blocks stale without refetching inactive ones', () => {
        const { client, calls } = spyClient()
        invalidateForLiveReconnect(client)

        expect(calls).toHaveLength(2)
        expect(calls[1].filters).toEqual({ queryKey: ['operations'] })
        expect(calls[1].options).toEqual({ cancelRefetch: false })
    })
})
