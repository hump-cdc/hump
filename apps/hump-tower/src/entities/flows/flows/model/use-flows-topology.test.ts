import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { cleanup, waitFor } from '@testing-library/react'

import { fakeApi, json } from '@/shared/testing/api.fake'
import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { metricsResponse } from '@/shared/testing/fixtures.fake'
import { renderHookWithProviders } from '@/shared/testing/render.fake'

import { useFlowsTopology } from './use-flows-topology'

beforeAll(registerDom)
afterAll(unregisterDom)
afterEach(cleanup)

describe('useFlowsTopology', () => {
    test('is empty until the metrics arrive, then maps their topology', async () => {
        fakeApi.serve({ 'POST /api/metrics': () => json(metricsResponse()) })

        const { result } = renderHookWithProviders(() => useFlowsTopology())

        expect(result.current).toEqual({ tables: [], sinks: [], edges: [], detailedSinks: [], detailedEdges: [] })
        await waitFor(() => expect(result.current.tables.map(({ id }) => id)).toEqual(['public.users']))
        expect(result.current.sinks.map(({ id }) => id)).toEqual(['http|api.example.com', 'kafka|kafka:9092'])
        expect(result.current.detailedEdges.map(({ id }) => id)).toEqual(['public.users>kafka|kafka:9092>users'])
    })
})
