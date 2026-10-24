import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { cleanup, waitFor } from '@testing-library/react'

import { fakeApi, json } from '@/shared/testing/api.fake'
import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { metricsResponse } from '@/shared/testing/fixtures.fake'
import { renderHookWithProviders, renderWithProviders } from '@/shared/testing/render.fake'

import { Failed } from '../index'
import { useFailed } from '../model/use-failed'

beforeAll(registerDom)
afterAll(unregisterDom)
afterEach(cleanup)

describe('useFailed', () => {
    test('sums the failed counts over the charted window', async () => {
        fakeApi.serve({ 'POST /api/metrics': () => json(metricsResponse()) })

        const { result } = renderHookWithProviders(() => useFailed())
        expect(result.current.value).toBeUndefined()
        await waitFor(() => expect(result.current.values).toHaveLength(20))
        expect(result.current.value).toBe(1)
        expect(result.current.values.at(-2)?.value).toBe(1)
    })
})

describe('Failed', () => {
    test('renders the Failed card', async () => {
        fakeApi.serve({ 'POST /api/metrics': () => json(metricsResponse()) })

        const { findByText } = renderWithProviders(<Failed />)

        expect(await findByText('Failed')).toBeDefined()
        await waitFor(() => expect(fakeApi.calls().map(({ route }) => route)).toEqual(['POST /api/metrics']))
    })
})
