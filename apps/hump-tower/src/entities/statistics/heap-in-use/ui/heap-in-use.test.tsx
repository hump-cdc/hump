import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { cleanup, waitFor } from '@testing-library/react'

import { fakeApi, json } from '@/shared/testing/api.fake'
import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { metricsResponse } from '@/shared/testing/fixtures.fake'
import { renderHookWithProviders, renderWithProviders } from '@/shared/testing/render.fake'

import { HeapInUse } from '../index'
import { useHeapInUse } from '../model/use-heap-in-use'

beforeAll(registerDom)
afterAll(unregisterDom)
afterEach(cleanup)

describe('useHeapInUse', () => {
    test('reports the newest heap reading in megabytes', async () => {
        fakeApi.serve({ 'POST /api/metrics': () => json(metricsResponse()) })

        const { result } = renderHookWithProviders(() => useHeapInUse())
        expect(result.current.value).toBeUndefined()
        await waitFor(() => expect(result.current.values).toHaveLength(20))
        expect(result.current.value).toBe(2)
        expect(result.current.values.at(-2)?.heapInUse).toBe(1)
    })
})

describe('HeapInUse', () => {
    test('renders the Memory card', async () => {
        fakeApi.serve({ 'POST /api/metrics': () => json(metricsResponse()) })

        const { findByText } = renderWithProviders(<HeapInUse />)

        expect(await findByText('Memory')).toBeDefined()
        await waitFor(() => expect(fakeApi.calls().map(({ route }) => route)).toEqual(['POST /api/metrics']))
    })
})
