import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { cleanup, waitFor } from '@testing-library/react'

import { fakeApi, json } from '@/shared/testing/api.fake'
import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { metricsResponse } from '@/shared/testing/fixtures.fake'
import { renderHookWithProviders, renderWithProviders } from '@/shared/testing/render.fake'

import { Throughput } from '../index'
import { useThroughput } from '../model/use-throughput'

beforeAll(registerDom)
afterAll(unregisterDom)
afterEach(cleanup)

describe('useThroughput', () => {
    test('rounds the rate of every bucket', async () => {
        fakeApi.serve({ 'POST /api/metrics': () => json(metricsResponse()) })

        const { result } = renderHookWithProviders(() => useThroughput())
        expect(result.current.values).toEqual([])
        await waitFor(() => expect(result.current.values).toHaveLength(20))
        expect(result.current.values.at(-2)?.eventsPerSec).toBe(1.23)
        expect(result.current.values.at(-1)?.eventsPerSec).toBe(2)
    })
})

describe('Throughput', () => {
    test('renders the Throughput card', async () => {
        fakeApi.serve({ 'POST /api/metrics': () => json(metricsResponse()) })

        const { findByText } = renderWithProviders(<Throughput />)

        expect(await findByText('Throughput')).toBeDefined()
        await waitFor(() => expect(fakeApi.calls().map(({ route }) => route)).toEqual(['POST /api/metrics']))
    })
})
