import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { cleanup, waitFor } from '@testing-library/react'

import { fakeApi, json } from '@/shared/testing/api.fake'
import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { metricsResponse } from '@/shared/testing/fixtures.fake'
import { renderHookWithProviders, renderWithProviders } from '@/shared/testing/render.fake'

import { SinkLatency } from '../index'
import { useSinkLatency } from '../model/use-sink-latency'

beforeAll(registerDom)
afterAll(unregisterDom)
afterEach(cleanup)

describe('useSinkLatency', () => {
    test('charts the rounded p95 of every sink, 0 without an estimate', async () => {
        fakeApi.serve({ 'POST /api/metrics': () => json(metricsResponse()) })

        const { result } = renderHookWithProviders(() => useSinkLatency())
        expect(result.current.series).toEqual([])
        await waitFor(() => expect(result.current.values).toHaveLength(20))
        expect(result.current.series.map(({ dataKey }) => dataKey)).toEqual([
            'http',
            'kafka',
            'nats',
            'rabbitmq',
            'redis',
        ])
        expect(result.current.values.at(-2)).toMatchObject({ kafka: 1.23, http: 0 })
        expect(result.current.values.at(-1)).toMatchObject({ kafka: 0, http: 5 })
    })
})

describe('SinkLatency', () => {
    test('renders the Sink latency card', async () => {
        fakeApi.serve({ 'POST /api/metrics': () => json(metricsResponse()) })

        const { findByText } = renderWithProviders(<SinkLatency />)

        expect(await findByText('Sink latency')).toBeDefined()
        await waitFor(() => expect(fakeApi.calls().map(({ route }) => route)).toEqual(['POST /api/metrics']))
    })
})
