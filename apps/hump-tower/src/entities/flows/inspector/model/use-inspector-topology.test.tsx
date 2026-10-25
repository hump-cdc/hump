import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { cleanup, waitFor } from '@testing-library/react'

import { fakeApi, json } from '@/shared/testing/api.fake'
import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { metricsResponse } from '@/shared/testing/fixtures.fake'
import { renderHookWithProviders } from '@/shared/testing/render.fake'

import { useInspectorTopology } from './use-inspector-topology'

beforeAll(registerDom)
afterAll(unregisterDom)
afterEach(cleanup)

describe('useInspectorTopology', () => {
    test('is undefined until the metrics arrive, then returns their topology', async () => {
        fakeApi.serve({ 'POST /api/metrics': () => json(metricsResponse()) })

        const { result } = renderHookWithProviders(() => useInspectorTopology())

        expect(result.current).toBeUndefined()
        await waitFor(() => expect(result.current?.tables.map(({ name }) => name)).toEqual(['public.users']))
        expect(result.current?.sinks.map(({ type }) => type)).toEqual(['http', 'kafka'])
    })
})
